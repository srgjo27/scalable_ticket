package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/srgjo27/scalable_ticket/internal/core/domain"
)

type SeatRepository interface {
	GetByID(ctx context.Context, seatID uuid.UUID) (*domain.Seat, error)
	GetAvailableSeatsByEvent(ctx context.Context, eventID uuid.UUID) ([]domain.Seat, error)
	LockSeat(ctx context.Context, seatID uuid.UUID, bookingID uuid.UUID, currentVersion int) error
	UnlockSeat(ctx context.Context, seatID uuid.UUID) error
}

type BookingRepository interface {
	CreateBooking(ctx context.Context, booking *domain.Booking) error
	UpdateStatus(ctx context.Context, bookingID uuid.UUID, status domain.BookingStatus) error
	GetExpiredBookings(ctx context.Context) ([]uuid.UUID, error)
	CancelBooking(ctx context.Context, bookingID uuid.UUID) error
}
