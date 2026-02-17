package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/srgjo27/scalable_ticket/internal/core/domain"
)

type SeatRepository struct {
	db *sql.DB
}

func NewSeatRepository(db *sql.DB) *SeatRepository {
	return &SeatRepository{db: db}
}

func (r *SeatRepository) GetByID(ctx context.Context, seatID uuid.UUID) (*domain.Seat, error) {
	query := `
	SELECT id, event_id, tier_id, section, row_number, seat_number, status, version, locked_by_booking_id, locked_at
	FROM event_seats
	WHERE id = $1
	`

	var seat domain.Seat
	var lockedBy sql.NullString
	var lockedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, seatID).Scan(
		&seat.ID,
		&seat.EventID,
		&seat.TierID,
		&seat.Section,
		&seat.RowNumber,
		&seat.SeatNumber,
		&seat.Status,
		&seat.Version,
		&lockedBy,
		&lockedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("seat not found")
		}

		return nil, err
	}

	if lockedBy.Valid && lockedBy.String != "" {
		uid, _ := uuid.Parse(lockedBy.String)
		seat.LockedByBookingID = &uid
	}

	if lockedAt.Valid {
		seat.LockedAt = &lockedAt.Time
	}

	return &seat, nil
}

func (r *SeatRepository) GetAvailableSeatsByEvent(ctx context.Context, eventID uuid.UUID) ([]domain.Seat, error) {
	query := `
	SELECT id, section, row_number, seat_number, status, version
	FROM event_seats
	WHERE event_id = $1 AND status = 'AVAILABLE'
	`
	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var seats []domain.Seat
	for rows.Next() {
		var seat domain.Seat
		if err := rows.Scan(
			&seat.ID,
			&seat.Section,
			&seat.RowNumber,
			&seat.SeatNumber,
			&seat.Status,
			&seat.Version,
		); err != nil {
			return nil, err
		}

		seats = append(seats, seat)
	}

	return seats, nil
}

func (r *SeatRepository) LockSeat(ctx context.Context, seatID uuid.UUID, bookingID uuid.UUID, currentVersion int) error {
	query := `
	UPDATE event_seats
	SET status = $1,
		locked_by_booking_id = $2,
		locked_at = $3,
		version = version + 1
	WHERE id = $4 AND version = $5 AND status = 'AVAILABLE'
	`

	result, err := r.db.ExecContext(ctx, query, domain.SeatLocked, bookingID, time.Now(), seatID, currentVersion)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("optimistic lock failed: seat was modified by another transaction")
	}

	return nil
}

func (r *SeatRepository) UnlockSeat(ctx context.Context, seatID uuid.UUID) error {
	query := `
	UPDATE event_seats
	SET status = 'AVAILABLE',
		locked_by_booking_id = NULL,
		locked_at = NULL,
		version = version + 1
	WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, seatID)

	return err
}
