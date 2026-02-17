package domain

import (
	"time"

	"github.com/google/uuid"
)

type SeatStatus string

const (
	SeatAvailable SeatStatus = "AVAILABLE"
	SeatLocked    SeatStatus = "LOCKED"
	SeatBooked    SeatStatus = "BOOKED"
	SeatSold      SeatStatus = "SOLD"
)

type Seat struct {
	ID                uuid.UUID
	EventID           uuid.UUID
	TierID            uuid.UUID
	Section           string
	RowNumber         string
	SeatNumber        string
	Status            SeatStatus
	Version           int
	LockedByBookingID *uuid.UUID
	LockedAt          *time.Time
}

func (s *Seat) IsAvailable() bool {
	return s.Status == SeatAvailable
}
