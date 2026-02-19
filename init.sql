CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE seat_status AS ENUM ('AVAILABLE', 'LOCKED', 'BOOKED', 'UNAVAILABLE');
CREATE TYPE booking_status AS ENUM ('PENDING', 'CONFIRMED', 'CANCELLED', 'EXPIRED', 'FAILED');

CREATE TABLE IF NOT EXISTS venues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    address TEXT,
    timezone VARCHAR(50) DEFAULT 'Asia/Jakarta',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    venue_id UUID REFERENCES venues(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pricing_tiers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS event_seats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    tier_id UUID REFERENCES pricing_tiers(id),
    section VARCHAR(50),
    row_number VARCHAR(10),
    seat_number VARCHAR(10),
    status seat_status DEFAULT 'AVAILABLE',
    version INT DEFAULT 1,
    locked_by_booking_id UUID,
    locked_at TIMESTAMPTZ,
    CONSTRAINT unique_seat_per_event UNIQUE (event_id, section, row_number, seat_number)
);

CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    event_id UUID REFERENCES events(id),
    total_amount DECIMAL(10, 2) NOT NULL,
    status booking_status DEFAULT 'PENDING',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS booking_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    booking_id UUID REFERENCES bookings(id) ON DELETE CASCADE,
    seat_id UUID REFERENCES event_seats(id),
    price_at_booking DECIMAL(10, 2) NOT NULL
);

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    booking_id UUID REFERENCES bookings(id),
    amount DECIMAL(10, 2) NOT NULL,
    payment_method VARCHAR(50),
    provider_transaction_id VARCHAR(255),
    status VARCHAR(50) DEFAULT 'SUCCESS',
    paid_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS seat_audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    seat_id UUID NOT NULL,
    old_status seat_status,
    new_status seat_status,
    changed_by_user_id UUID,
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_seats_event_status ON event_seats(event_id, status);
CREATE INDEX IF NOT EXISTS idx_bookings_status_expires ON bookings(status, expires_at);
CREATE INDEX IF NOT EXISTS idx_seats_booking_lock ON event_seats(locked_by_booking_id);

INSERT INTO venues (id, name, address) VALUES 
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Stadion GBK', 'Jl. Pintu Satu Senayan, Jakarta');

INSERT INTO events (id, venue_id, name, start_time, end_time) VALUES 
('b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Coldplay Jakarta', NOW() + INTERVAL '1 month', NOW() + INTERVAL '1 month 4 hours');

INSERT INTO pricing_tiers (id, event_id, name, price) VALUES
('f1eebc99-9c0b-4ef8-bb6d-6bb9bd380a99', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'VIP', 5000000);

INSERT INTO event_seats (id, event_id, tier_id, section, row_number, seat_number, status, version) VALUES
('c1eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'f1eebc99-9c0b-4ef8-bb6d-6bb9bd380a99', 'A', '1', '1', 'AVAILABLE', 1),
('c1eebc99-9c0b-4ef8-bb6d-6bb9bd380a02', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'f1eebc99-9c0b-4ef8-bb6d-6bb9bd380a99', 'A', '1', '2', 'AVAILABLE', 1);