package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/example/booking-service/internal/model"
	"github.com/example/booking-service/internal/repository"
)

// Sentinel errors returned to handlers.
var (
	ErrSlotNotFound    = errors.New("slot not found")
	ErrSlotInPast      = errors.New("slot is in the past")
	ErrSlotBooked      = errors.New("slot already booked")
	ErrBookingNotFound = errors.New("booking not found")
	ErrForbidden       = errors.New("forbidden")
)

type BookingService struct {
	bookings    repository.BookingRepository
	slots       repository.SlotRepository
	conference  ConferenceService
}

func NewBookingService(
	bookings repository.BookingRepository,
	slots repository.SlotRepository,
	conference ConferenceService,
) *BookingService {
	return &BookingService{
		bookings:   bookings,
		slots:      slots,
		conference: conference,
	}
}

type CreateBookingInput struct {
	SlotID              uuid.UUID
	UserID              uuid.UUID
	CreateConferenceLink bool
}

func (s *BookingService) Create(ctx context.Context, inp CreateBookingInput) (*model.Booking, error) {
	// Verify slot exists
	slot, err := s.slots.GetByID(ctx, inp.SlotID)
	if err != nil {
		return nil, fmt.Errorf("get slot: %w", err)
	}
	if slot == nil {
		return nil, ErrSlotNotFound
	}

	// Reject past slots
	if slot.StartAt.Before(time.Now().UTC()) {
		return nil, ErrSlotInPast
	}

	booking := &model.Booking{
		ID:        uuid.New(),
		SlotID:    inp.SlotID,
		UserID:    inp.UserID,
		Status:    model.BookingStatusActive,
		CreatedAt: time.Now().UTC(),
	}

	// Optionally fetch conference link (best-effort — never blocks booking creation)
	if inp.CreateConferenceLink {
		link, linkErr := s.conference.CreateLink(ctx, booking.ID)
		if linkErr != nil {
			log.Printf("booking %s: conference link failed (best-effort): %v", booking.ID, linkErr)
		} else {
			booking.ConferenceLink = &link
		}
	}

	if err := s.bookings.Create(ctx, booking); err != nil {
		if errors.Is(err, repository.ErrSlotAlreadyBooked) {
			return nil, ErrSlotBooked
		}
		return nil, fmt.Errorf("persist booking: %w", err)
	}

	return booking, nil
}

// Cancel cancels the booking. Idempotent: returns current state if already cancelled.
// Returns ErrForbidden if the booking belongs to a different user.
func (s *BookingService) Cancel(ctx context.Context, bookingID, userID uuid.UUID) (*model.Booking, error) {
	booking, err := s.bookings.GetByID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("get booking: %w", err)
	}
	if booking == nil {
		return nil, ErrBookingNotFound
	}
	if booking.UserID != userID {
		return nil, ErrForbidden
	}

	// Idempotent — if already cancelled, just return current state
	if booking.Status == model.BookingStatusCancelled {
		return booking, nil
	}

	if err := s.bookings.Cancel(ctx, bookingID); err != nil {
		return nil, fmt.Errorf("cancel booking: %w", err)
	}
	booking.Status = model.BookingStatusCancelled
	return booking, nil
}

func (s *BookingService) GetMyBookings(ctx context.Context, userID uuid.UUID) ([]model.Booking, error) {
	bookings, err := s.bookings.ListByUserFuture(ctx, userID, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("list my bookings: %w", err)
	}
	return bookings, nil
}

func (s *BookingService) ListAll(ctx context.Context, page, pageSize int) ([]model.Booking, int, error) {
	total, err := s.bookings.CountAll(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count bookings: %w", err)
	}

	offset := (page - 1) * pageSize
	bookings, err := s.bookings.ListAll(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list bookings: %w", err)
	}
	return bookings, total, nil
}

func (s *BookingService) GetAvailableSlots(ctx context.Context, roomID uuid.UUID, date time.Time) ([]model.Slot, error) {
	slots, err := s.slots.GetAvailable(ctx, roomID, date)
	if err != nil {
		return nil, fmt.Errorf("get available slots: %w", err)
	}
	return slots, nil
}
