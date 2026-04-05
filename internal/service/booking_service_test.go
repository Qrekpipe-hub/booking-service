package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Qrekpipe-hub/booking-service/internal/model"
	"github.com/Qrekpipe-hub/booking-service/internal/service"
	"github.com/Qrekpipe-hub/booking-service/internal/service/mocks"
)

func futureSlot(roomID uuid.UUID) *model.Slot {
	start := time.Now().UTC().Add(1 * time.Hour)
	return &model.Slot{
		ID:      uuid.New(),
		RoomID:  roomID,
		StartAt: start,
		EndAt:   start.Add(30 * time.Minute),
	}
}

func pastSlot(roomID uuid.UUID) *model.Slot {
	start := time.Now().UTC().Add(-2 * time.Hour)
	return &model.Slot{
		ID:      uuid.New(),
		RoomID:  roomID,
		StartAt: start,
		EndAt:   start.Add(30 * time.Minute),
	}
}

func newBookingService(slotRepo *mocks.MockSlotRepo, bookingRepo *mocks.MockBookingRepo, conf service.ConferenceService) *service.BookingService {
	return service.NewBookingService(bookingRepo, slotRepo, conf)
}


func TestBookingCreate_Success(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	bookingRepo := mocks.NewMockBookingRepo()
	conf := &mocks.MockConferenceService{}

	roomID := uuid.New()
	slot := futureSlot(roomID)
	slotRepo.Slots[slot.ID] = slot

	svc := newBookingService(slotRepo, bookingRepo, conf)

	userID := uuid.New()
	booking, err := svc.Create(context.Background(), service.CreateBookingInput{
		SlotID: slot.ID,
		UserID: userID,
	})

	require.NoError(t, err)
	assert.Equal(t, slot.ID, booking.SlotID)
	assert.Equal(t, userID, booking.UserID)
	assert.Equal(t, model.BookingStatusActive, booking.Status)
	assert.Nil(t, booking.ConferenceLink)
}

func TestBookingCreate_SlotNotFound(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	bookingRepo := mocks.NewMockBookingRepo()
	svc := newBookingService(slotRepo, bookingRepo, &mocks.MockConferenceService{})

	_, err := svc.Create(context.Background(), service.CreateBookingInput{
		SlotID: uuid.New(),
		UserID: uuid.New(),
	})

	assert.ErrorIs(t, err, service.ErrSlotNotFound)
}

func TestBookingCreate_SlotInPast(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	bookingRepo := mocks.NewMockBookingRepo()
	svc := newBookingService(slotRepo, bookingRepo, &mocks.MockConferenceService{})

	roomID := uuid.New()
	slot := pastSlot(roomID)
	slotRepo.Slots[slot.ID] = slot

	_, err := svc.Create(context.Background(), service.CreateBookingInput{
		SlotID: slot.ID,
		UserID: uuid.New(),
	})

	assert.ErrorIs(t, err, service.ErrSlotInPast)
}

func TestBookingCreate_SlotAlreadyBooked(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	bookingRepo := mocks.NewMockBookingRepo()
	svc := newBookingService(slotRepo, bookingRepo, &mocks.MockConferenceService{})

	roomID := uuid.New()
	slot := futureSlot(roomID)
	slotRepo.Slots[slot.ID] = slot

	userID1 := uuid.New()
	userID2 := uuid.New()

	// First booking succeeds
	_, err := svc.Create(context.Background(), service.CreateBookingInput{SlotID: slot.ID, UserID: userID1})
	require.NoError(t, err)

	// Second booking on same slot fails
	_, err = svc.Create(context.Background(), service.CreateBookingInput{SlotID: slot.ID, UserID: userID2})
	assert.ErrorIs(t, err, service.ErrSlotBooked)
}

func TestBookingCreate_WithConferenceLinkSuccess(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	bookingRepo := mocks.NewMockBookingRepo()
	conf := &mocks.MockConferenceService{Link: "https://conf.internal/room/test-abc"}

	roomID := uuid.New()
	slot := futureSlot(roomID)
	slotRepo.Slots[slot.ID] = slot

	svc := newBookingService(slotRepo, bookingRepo, conf)

	booking, err := svc.Create(context.Background(), service.CreateBookingInput{
		SlotID:               slot.ID,
		UserID:               uuid.New(),
		CreateConferenceLink: true,
	})

	require.NoError(t, err)
	require.NotNil(t, booking.ConferenceLink)
	assert.Equal(t, "https://conf.internal/room/test-abc", *booking.ConferenceLink)
}

func TestBookingCreate_ConferenceLinkFailsBookingStillCreated(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	bookingRepo := mocks.NewMockBookingRepo()
	conf := &mocks.MockConferenceService{Err: assert.AnError} // conference service is down

	roomID := uuid.New()
	slot := futureSlot(roomID)
	slotRepo.Slots[slot.ID] = slot

	svc := newBookingService(slotRepo, bookingRepo, conf)

	// Бронь должна создаться даже при недоступности conference service
	booking, err := svc.Create(context.Background(), service.CreateBookingInput{
		SlotID:               slot.ID,
		UserID:               uuid.New(),
		CreateConferenceLink: true,
	})

	require.NoError(t, err)
	assert.Nil(t, booking.ConferenceLink)
	assert.Equal(t, model.BookingStatusActive, booking.Status)
}


func TestBookingCancel_Success(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	bookingRepo := mocks.NewMockBookingRepo()
	svc := newBookingService(slotRepo, bookingRepo, &mocks.MockConferenceService{})

	roomID := uuid.New()
	slot := futureSlot(roomID)
	slotRepo.Slots[slot.ID] = slot
	userID := uuid.New()

	booking, _ := svc.Create(context.Background(), service.CreateBookingInput{SlotID: slot.ID, UserID: userID})

	cancelled, err := svc.Cancel(context.Background(), booking.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, model.BookingStatusCancelled, cancelled.Status)
}

func TestBookingCancel_Idempotent(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	bookingRepo := mocks.NewMockBookingRepo()
	svc := newBookingService(slotRepo, bookingRepo, &mocks.MockConferenceService{})

	roomID := uuid.New()
	slot := futureSlot(roomID)
	slotRepo.Slots[slot.ID] = slot
	userID := uuid.New()

	booking, _ := svc.Create(context.Background(), service.CreateBookingInput{SlotID: slot.ID, UserID: userID})
	_, err1 := svc.Cancel(context.Background(), booking.ID, userID)
	_, err2 := svc.Cancel(context.Background(), booking.ID, userID) // second cancel

	require.NoError(t, err1)
	require.NoError(t, err2, "second cancel must be idempotent (no error)")
}

func TestBookingCancel_OtherUsersForbidden(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	bookingRepo := mocks.NewMockBookingRepo()
	svc := newBookingService(slotRepo, bookingRepo, &mocks.MockConferenceService{})

	roomID := uuid.New()
	slot := futureSlot(roomID)
	slotRepo.Slots[slot.ID] = slot
	ownerID := uuid.New()
	otherID := uuid.New()

	booking, _ := svc.Create(context.Background(), service.CreateBookingInput{SlotID: slot.ID, UserID: ownerID})

	_, err := svc.Cancel(context.Background(), booking.ID, otherID)
	assert.ErrorIs(t, err, service.ErrForbidden)
}

func TestBookingCancel_NotFound(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	bookingRepo := mocks.NewMockBookingRepo()
	svc := newBookingService(slotRepo, bookingRepo, &mocks.MockConferenceService{})

	_, err := svc.Cancel(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, service.ErrBookingNotFound)
}


func TestBookingCancelThenRebook(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	bookingRepo := mocks.NewMockBookingRepo()
	svc := newBookingService(slotRepo, bookingRepo, &mocks.MockConferenceService{})

	roomID := uuid.New()
	slot := futureSlot(roomID)
	slotRepo.Slots[slot.ID] = slot
	user1 := uuid.New()
	user2 := uuid.New()

	b1, _ := svc.Create(context.Background(), service.CreateBookingInput{SlotID: slot.ID, UserID: user1})
	_, _ = svc.Cancel(context.Background(), b1.ID, user1)

	// Now user2 should be able to book the same slot
	b2, err := svc.Create(context.Background(), service.CreateBookingInput{SlotID: slot.ID, UserID: user2})
	require.NoError(t, err)
	assert.Equal(t, model.BookingStatusActive, b2.Status)
}
