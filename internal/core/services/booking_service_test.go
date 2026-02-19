package services_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/srgjo27/scalable_ticket/internal/core/domain"
	"github.com/srgjo27/scalable_ticket/internal/core/ports/mocks"
	"github.com/srgjo27/scalable_ticket/internal/core/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateBooking_Success(t *testing.T) {
	mockSeatRepo := mocks.NewSeatRepository(t)
	mockBookingRepo := mocks.NewBookingRepository(t)

	db, mockRedis := redismock.NewClientMock()

	service := services.NewBookingService(mockSeatRepo, mockBookingRepo, db)

	ctx := context.Background()
	userID := uuid.New()
	eventID := uuid.New()
	seatID := uuid.New()

	mockSeat := &domain.Seat{
		ID:         seatID,
		EventID:    eventID,
		Status:     domain.SeatAvailable,
		Version:    1,
		SeatNumber: "A1",
	}

	req := services.CreateBookingRequest{
		UserID:  userID.String(),
		EventID: eventID.String(),
		SeatIDs: []string{seatID.String()},
	}

	mockSeatRepo.On("GetByID", ctx, seatID).Return(mockSeat, nil)
	mockSeatRepo.On("LockSeat", ctx, seatID, mock.AnythingOfType("uuid.UUID"), 1).Return(nil)
	mockBookingRepo.On("CreateBooking", ctx, mock.AnythingOfType("*domain.Booking")).Return(nil)

	cacheKey := fmt.Sprintf("seats:%s", eventID.String())
	mockRedis.ExpectDel(cacheKey).SetVal(1)

	resp, err := service.CreateBooking(ctx, req)

	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.Equal(t, 100000.0, resp.TotalAmount)
	}

	if err := mockRedis.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCreateBooking_Fail_SeatLocked(t *testing.T) {
	mockSeatRepo := mocks.NewSeatRepository(t)
	mockBookingRepo := mocks.NewBookingRepository(t)
	db, _ := redismock.NewClientMock()

	service := services.NewBookingService(mockSeatRepo, mockBookingRepo, db)

	ctx := context.Background()
	seatID := uuid.New()
	eventID := uuid.New()

	mockSeat := &domain.Seat{
		ID:      seatID,
		EventID: eventID,
		Status:  domain.SeatAvailable,
		Version: 1,
	}

	req := services.CreateBookingRequest{
		UserID:  uuid.New().String(),
		EventID: eventID.String(),
		SeatIDs: []string{seatID.String()},
	}

	mockSeatRepo.On("GetByID", ctx, seatID).Return(mockSeat, nil)
	mockSeatRepo.On("LockSeat", ctx, seatID, mock.Anything, 1).Return(errors.New("optimistic lock failed"))

	resp, err := service.CreateBooking(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to lock seat")
}
