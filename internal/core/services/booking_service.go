package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/srgjo27/scalable_ticket/internal/core/domain"
	"github.com/srgjo27/scalable_ticket/internal/core/ports"
)

type CreateBookingRequest struct {
	UserID  string   `json:"user_id"`
	EventID string   `json:"event_id"`
	SeatIDs []string `json:"seat_ids"`
}

type CreateBookingResponse struct {
	BookingID   string  `json:"booking_id"`
	TotalAmount float64 `json:"total_amount"`
	Status      string  `json:"status"`
	ExpiresAt   string  `json:"expires_at"`
}

type BookingService struct {
	seatRepo    ports.SeatRepository
	bookingRepo ports.BookingRepository
}

func NewBookingService(seatRepo ports.SeatRepository, bookingRepo ports.BookingRepository) *BookingService {
	return &BookingService{
		seatRepo:    seatRepo,
		bookingRepo: bookingRepo,
	}
}

func (s *BookingService) CreateBooking(ctx context.Context, req CreateBookingRequest) (*CreateBookingResponse, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, errors.New("invalid user id")
	}

	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		return nil, errors.New("invalid event id")
	}

	if len(req.SeatIDs) == 0 {
		return nil, errors.New("no seats selected")
	}

	bookingID := uuid.New()

	var lockedSeatIDs []uuid.UUID
	var totalAmount float64
	var bookingItems []domain.BookingItem

	for _, seatIDStr := range req.SeatIDs {
		seatID, _ := uuid.Parse(seatIDStr)

		seat, err := s.seatRepo.GetByID(ctx, seatID)
		if err != nil {
			s.rollbackLocks(ctx, lockedSeatIDs)
			return nil, fmt.Errorf("seat not found: %s", seatIDStr)
		}

		if !seat.IsAvailable() {
			s.rollbackLocks(ctx, lockedSeatIDs)
			return nil, fmt.Errorf("seat %s is not available", seat.SeatNumber)
		}

		if seat.EventID != eventID {
			s.rollbackLocks(ctx, lockedSeatIDs)
			return nil, errors.New("seat does not belong to this event")
		}

		err = s.seatRepo.LockSeat(ctx, seat.ID, bookingID, seat.Version)
		if err != nil {
			s.rollbackLocks(ctx, lockedSeatIDs)
			return nil, fmt.Errorf("failed to lock seat %s: maybe taken by another user", seat.SeatNumber)
		}

		lockedSeatIDs = append(lockedSeatIDs, seat.ID)

		seatPrice := 100000.00
		totalAmount += seatPrice

		bookingItems = append(bookingItems, domain.BookingItem{
			ID:             uuid.New(),
			BookingID:      bookingID,
			SeatID:         seat.ID,
			PriceAtBooking: seatPrice,
		})
	}

	expiresAt := time.Now().Add(10 * time.Minute)

	newBooking := &domain.Booking{
		ID:          bookingID,
		UserID:      userID,
		EventID:     eventID,
		TotalAmount: totalAmount,
		Status:      domain.BookingPending,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		Items:       bookingItems,
	}

	err = s.bookingRepo.CreateBooking(ctx, newBooking)
	if err != nil {
		s.rollbackLocks(ctx, lockedSeatIDs)
		return nil, errors.New("internal server error: failed to create booking")
	}

	return &CreateBookingResponse{
		BookingID:   bookingID.String(),
		TotalAmount: totalAmount,
		Status:      string(domain.BookingPending),
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}, nil
}

func (s *BookingService) rollbackLocks(ctx context.Context, seatIDs []uuid.UUID) {
	for _, id := range seatIDs {
		_ = s.seatRepo.UnlockSeat(ctx, id)
	}
}

func (s *BookingService) RunBackgroundCleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Background Worker started: Checking expired bookings every 1 minute...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Background Worker stopped.")
			return
		case <-ticker.C:
			s.processExpiredBookings(ctx)
		}
	}
}

func (s *BookingService) processExpiredBookings(ctx context.Context) {
	ids, err := s.bookingRepo.GetExpiredBookings(ctx)
	if err != nil {
		log.Printf("Error fetching expired bookings: %v", err)
		return
	}

	if len(ids) == 0 {
		return
	}

	log.Printf("Found %d expired bookings. Cleaning up...", len(ids))

	for _, id := range ids {
		if err := s.bookingRepo.CancelBooking(ctx, id); err != nil {
			log.Printf("Failed to cancel booking %s: %v", id, err)
		} else {
			log.Printf("Booking %s expired and seats released.", id)
		}
	}
}
