package domain

import (
	"time"

	"github.com/google/uuid"
)

type BookingStatus string

const (
	BookingPending   BookingStatus = "PENDING"
	BookingConfirmed BookingStatus = "CONFIRMED"
	BookingExpired   BookingStatus = "EXPIRED"
	BookingCancelled BookingStatus = "CANCELLED"
)

type Booking struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	EventID     uuid.UUID
	TotalAmount float64
	Status      BookingStatus
	CreatedAt   time.Time
	ExpiresAt   time.Time
	ConfirmedAt *time.Time
	Items       []BookingItem
}

type BookingItem struct {
	ID             uuid.UUID
	BookingID      uuid.UUID
	SeatID         uuid.UUID
	PriceAtBooking float64
}
