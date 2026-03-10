# 🎟️ Scalable Ticket — Backend Service

A high-performance, race-condition-safe ticket booking backend built with **Go**, following **Hexagonal (Clean) Architecture** principles. Designed to handle concurrent seat reservations reliably using optimistic locking at the database level and Redis for caching.

---

## 📌 Overview

**Scalable Ticket** is a personal portfolio project that simulates a real-world, high-traffic concert ticketing system — think Coldplay Jakarta selling out in minutes. The core challenge solved here is preventing **double booking** under concurrent load without relying on application-level locks or Kafka/queue systems, keeping the stack lean and the logic correct.

### Goals
- Handle concurrent booking requests safely using **optimistic locking**
- Keep architecture clean and testable via **dependency injection** and **interface-driven ports**
- Automatically release seats from expired bookings via a **background worker**
- Reduce database read pressure with **Redis cache** for available seat queries
- Fully containerized with **Docker Compose** for zero-friction local setup

---

## 🏗️ Architecture

This project follows the **Hexagonal Architecture** (Ports & Adapters) pattern, organizing code into clearly separated layers:

```
scalable-ticket/
├── cmd/
│   └── api/
│       └── main.go              # Application entrypoint (wiring & server)
├── internal/
│   ├── core/                    # Inner hexagon — business logic
│   │   ├── domain/              # Pure domain entities & business rules
│   │   │   ├── booking.go       # Booking, BookingItem, BookingStatus
│   │   │   └── seat.go          # Seat, SeatStatus, IsAvailable()
│   │   ├── ports/               # Interface contracts (driven & driving)
│   │   │   ├── repository.go    # SeatRepository & BookingRepository interfaces
│   │   │   └── mocks/           # Auto-generated mocks for unit testing
│   │   └── services/            # Use-case implementations
│   │       ├── booking_service.go
│   │       └── booking_service_test.go
│   ├── adapter/                 # Outer hexagon — infrastructure adapters
│   │   ├── handler/             # HTTP handlers (driving adapter)
│   │   │   └── booking_handler.go
│   │   └── repository/          # Database adapters (driven adapter)
│   │       └── postgres/
│   │           ├── seat_repository.go
│   │           └── booking_repository.go
│   └── platform/                # Cross-cutting platform concerns
│       └── database/
│           └── postgres.go      # DB connection with retry logic
├── init.sql                     # PostgreSQL schema + seed data
├── Dockerfile                   # Multi-stage build
└── docker-compose.yml           # Full stack: API + PostgreSQL + Redis
```

### Layer Responsibilities

| Layer | Package | Role |
|---|---|---|
| **Domain** | `core/domain` | Pure entities, value objects, business rules — zero external dependencies |
| **Ports** | `core/ports` | Go interfaces that decouple the service layer from infrastructure |
| **Services** | `core/services` | Orchestrates use-cases: booking creation, availability, cleanup |
| **Adapters (in)** | `adapter/handler` | HTTP request parsing, response serialization |
| **Adapters (out)** | `adapter/repository/postgres` | SQL queries against PostgreSQL |
| **Platform** | `platform/database` | DB connection management with retry logic |

---

## 🛠️ Tech Stack

| Technology | Version | Purpose |
|---|---|---|
| **Go** | 1.23 | Primary language |
| **PostgreSQL** | 15 | Primary relational datastore |
| **Redis** | 7 | Read-through cache for available seats |
| **`lib/pq`** | v1.11.2 | PostgreSQL driver |
| **`go-redis/v9`** | v9.18 | Redis client |
| **`google/uuid`** | v1.6.0 | UUID generation for all entity IDs |
| **`stretchr/testify`** | v1.11 | Assertions & mocking framework for unit tests |
| **`go-redis/redismock`** | v9.2.0 | Redis mock for testing |
| **Docker / Docker Compose** | — | Containerization & local orchestration |

> **No web framework used.** The HTTP server is built entirely on Go's standard library `net/http`, keeping dependencies minimal and performance predictable.

---

## ✨ Key Features

### 1. Optimistic Locking for Concurrent Seat Reservation
Each `event_seats` row carries a `version` integer column. When locking a seat, the query performs:

```sql
UPDATE event_seats
SET status = 'LOCKED', locked_by_booking_id = $2, locked_at = $3, version = version + 1
WHERE id = $4 AND version = $5 AND status = 'AVAILABLE'
```

If another transaction modified the row first, `rows_affected = 0` and the booking atomically fails — **no pessimistic locks, no deadlocks**.

### 2. Automatic Lock Rollback on Partial Failure
When booking multiple seats, if any seat fails to lock, all previously locked seats in the same request are **rolled back immediately** before returning an error to the client.

### 3. Redis Cache — Available Seats
`GET /seats?event_id=...` first checks Redis (`seats:{event_id}`). On a cache miss it queries PostgreSQL and writes the result back with a 1-minute TTL. The cache is **invalidated** on every successful booking creation.

### 4. Background Cleanup Worker
A goroutine runs every **1 minute** and queries for bookings that have `status = 'PENDING'` and `expires_at < NOW()`. For each expired booking:
- The booking status is updated to `EXPIRED`
- All associated seats are unlocked (`status → AVAILABLE`, `locked_by_booking_id → NULL`)

These operations run inside a **database transaction** to ensure consistency.

### 5. Graceful Shutdown
The server listens for `SIGINT` / `SIGTERM` and performs a graceful shutdown with a **5-second drain timeout**, ensuring in-flight requests complete before the process exits.

---

## 🗃️ Database Schema

```
venues ──< events ──< pricing_tiers
                  ──< event_seats ──< booking_items >── bookings
                                                          │
                                                       payments
                                                   seat_audit_logs
```

**Key design decisions:**
- All primary keys are `UUID` (via PostgreSQL `uuid-ossp` extension)
- `event_seats.version` enables optimistic concurrency control
- `bookings.expires_at` drives the background expiry cleanup
- Indexes on `(event_id, status)`, `(status, expires_at)`, and `(locked_by_booking_id)` for query performance

---

## 🔌 API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/bookings` | Create a new booking for one or more seats |
| `GET` | `/seats?event_id={uuid}` | List available seats for an event (cached) |

### `POST /bookings` — Request Body
```json
{
  "user_id": "uuid",
  "event_id": "uuid",
  "seat_ids": ["uuid", "uuid"]
}
```

### `POST /bookings` — Success Response `201 Created`
```json
{
  "booking_id": "uuid",
  "total_amount": 100000.00,
  "status": "PENDING",
  "expires_at": "2026-03-10T15:22:06+07:00"
}
```

### Error Responses

| Status | Scenario |
|---|---|
| `400 Bad Request` | Invalid UUID, empty seat list, malformed JSON |
| `405 Method Not Allowed` | Wrong HTTP method |
| `409 Conflict` | Seat not found or already taken (optimistic lock failed) |
| `500 Internal Server Error` | Database or unexpected error |

---

## 🚀 Getting Started

### Prerequisites
- [Docker](https://www.docker.com/get-started) & Docker Compose v2+
- [Go 1.23+](https://go.dev/dl/) *(for local development only)*

### 1. Clone the Repository
```bash
git clone https://github.com/srgjo27/scalable_ticket.git
cd scalable_ticket
```

### 2. Configure Environment Variables
Create a `.env` file in the project root:
```env
DB_HOST=db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=secret
DB_NAME=scalable_ticket
REDIS_HOST=redis
REDIS_PORT=6379
```

### 3. Run with Docker Compose
```bash
docker compose up --build
```

This will spin up:
- `ticket-db` — PostgreSQL 15 (schema auto-applied via `init.sql`)
- `ticket-redis` — Redis 7
- `ticket-api` — Go API server on port `8080`

### 4. Test the API

**Get available seats** (uses the seeded event from `init.sql`):
```bash
curl "http://localhost:8080/seats?event_id=b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22"
```

**Create a booking:**
```bash
curl -X POST http://localhost:8080/bookings \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    "event_id": "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
    "seat_ids": ["c1eebc99-9c0b-4ef8-bb6d-6bb9bd380a01"]
  }'
```

---

## 🧪 Running Tests

Unit tests cover the core business logic with mocked repositories and a mocked Redis client — no real infrastructure required:

```bash
go test ./internal/core/services/... -v
```

**Test coverage:**
- `TestCreateBooking_Success` — happy path: seat locked, booking persisted, Redis cache invalidated
- `TestCreateBooking_Fail_SeatLocked` — concurrent conflict: optimistic lock rejection propagates correctly

---

## 🔧 Local Development (Without Docker)

Ensure PostgreSQL and Redis are running locally, then:

```bash
# Apply database schema
psql -U postgres -d scalable_ticket -f init.sql

# Set environment variables
export DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=secret DB_NAME=scalable_ticket
export REDIS_HOST=localhost REDIS_PORT=6379

# Run the server
go run cmd/api/main.go
```

---

## 🔮 Potential Future Enhancements

- [ ] Payment confirmation flow (`PENDING → CONFIRMED`)
- [ ] JWT-based authentication middleware
- [ ] Event-driven architecture with message queue (e.g., NATS / RabbitMQ) for payment processing
- [ ] Prometheus metrics endpoint for observability
- [ ] Rate limiting per user to prevent abuse
- [ ] Structured logging (e.g., `zerolog` or `slog`)

---

## 📄 License

This project is open source and available under the [MIT License](LICENSE).
