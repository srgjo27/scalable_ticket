package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/srgjo27/scalable_ticket/internal/core/domain"
)

type BookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) CreateBooking(ctx context.Context, booking *domain.Booking) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	queryHeader := `
	INSERT INTO bookings (id, user_id, event_id, total_amount, status, created_at, expires_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = tx.ExecContext(ctx, queryHeader, booking.ID, booking.UserID, booking.EventID, booking.TotalAmount, booking.Status, booking.CreatedAt, booking.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to insert booking header: %w", err)
	}

	queryItem := `
	INSERT INTO booking_items (id, booking_id, seat_id, price_at_booking)
	VALUES ($1, $2, $3, $4)
	`

	stmt, err := tx.PrepareContext(ctx, queryItem)
	if err != nil {
		return fmt.Errorf("failed to prepare item statement: %w", err)
	}

	defer stmt.Close()

	for _, item := range booking.Items {
		_, err := stmt.ExecContext(ctx, item.ID, item.BookingID, item.SeatID, item.PriceAtBooking)
		if err != nil {
			return fmt.Errorf("failed to insert booking item seat %s: %w", item.SeatID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *BookingRepository) UpdateStatus(ctx context.Context, bookingID uuid.UUID, status domain.BookingStatus) error {
	query := `
	UPDATE bookings
	SET status = $1, confirmed_at = $2
	WHERE id = $2
	`

	var confirmedAt *time.Time
	if status == domain.BookingConfirmed {
		now := time.Now()
		confirmedAt = &now
	}

	_, err := r.db.ExecContext(ctx, query, status, confirmedAt, bookingID)
	if err != nil {
		return err
	}

	return nil
}

func (r *BookingRepository) GetExpiredBookings(ctx context.Context) ([]uuid.UUID, error) {
	query := `
	SELECT id FROM bookings
	WHERE status = 'PENDING' AND expires_at < NOW()
	LIMIT 100
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, nil
}

func (r *BookingRepository) CancelBooking(ctx context.Context, bookingID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `UPDATE bookings SET status = 'EXPIRED' WHERE id = $1`, bookingID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
	UPDATE event_seats
	SET status = 'AVAILABLE',
		locked_by_booking_id = NULL,
		locked_at = NULL,
		version = version + 1
	WHERE locked_by_booking_id = $1
	`, bookingID)

	if err != nil {
		return err
	}

	return tx.Commit()
}
