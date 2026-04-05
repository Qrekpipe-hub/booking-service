package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/example/booking-service/internal/model"
	"github.com/example/booking-service/internal/repository"
)

// ── Slot Repository Mock ──────────────────────────────────────────

type MockSlotRepo struct {
	Slots         map[uuid.UUID]*model.Slot
	BulkInsertErr error
	MaxDate       *time.Time
	MaxDateErr    error
	Schedules     []model.Schedule
}

func NewMockSlotRepo() *MockSlotRepo {
	return &MockSlotRepo{Slots: make(map[uuid.UUID]*model.Slot)}
}

func (m *MockSlotRepo) BulkInsert(_ context.Context, slots []model.Slot) error {
	if m.BulkInsertErr != nil {
		return m.BulkInsertErr
	}
	for i := range slots {
		s := slots[i]
		m.Slots[s.ID] = &s
	}
	return nil
}

func (m *MockSlotRepo) GetAvailable(_ context.Context, roomID uuid.UUID, date time.Time) ([]model.Slot, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	var result []model.Slot
	for _, s := range m.Slots {
		if s.RoomID == roomID && !s.StartAt.Before(dayStart) && s.StartAt.Before(dayEnd) {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *MockSlotRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Slot, error) {
	s, ok := m.Slots[id]
	if !ok {
		return nil, nil
	}
	return s, nil
}

func (m *MockSlotRepo) MaxSlotDate(_ context.Context, _ uuid.UUID) (*time.Time, error) {
	return m.MaxDate, m.MaxDateErr
}

func (m *MockSlotRepo) ListSchedulesWithRooms(_ context.Context) ([]model.Schedule, error) {
	return m.Schedules, nil
}

// ── Booking Repository Mock ───────────────────────────────────────

type MockBookingRepo struct {
	Bookings      map[uuid.UUID]*model.Booking
	CreateErr     error
}

func NewMockBookingRepo() *MockBookingRepo {
	return &MockBookingRepo{Bookings: make(map[uuid.UUID]*model.Booking)}
}

func (m *MockBookingRepo) Create(_ context.Context, b *model.Booking) error {
	if m.CreateErr != nil {
		return m.CreateErr
	}
	// Check for existing active booking on same slot
	for _, existing := range m.Bookings {
		if existing.SlotID == b.SlotID && existing.Status == model.BookingStatusActive {
			return repository.ErrSlotAlreadyBooked
		}
	}
	cp := *b
	m.Bookings[b.ID] = &cp
	return nil
}

func (m *MockBookingRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Booking, error) {
	b, ok := m.Bookings[id]
	if !ok {
		return nil, nil
	}
	return b, nil
}

func (m *MockBookingRepo) Cancel(_ context.Context, id uuid.UUID) error {
	if b, ok := m.Bookings[id]; ok {
		b.Status = model.BookingStatusCancelled
	}
	return nil
}

func (m *MockBookingRepo) ListByUserFuture(_ context.Context, userID uuid.UUID, now time.Time) ([]model.Booking, error) {
	var result []model.Booking
	for _, b := range m.Bookings {
		if b.UserID == userID {
			result = append(result, *b)
		}
	}
	return result, nil
}

func (m *MockBookingRepo) ListAll(_ context.Context, limit, offset int) ([]model.Booking, error) {
	var all []model.Booking
	for _, b := range m.Bookings {
		all = append(all, *b)
	}
	if offset >= len(all) {
		return []model.Booking{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (m *MockBookingRepo) CountAll(_ context.Context) (int, error) {
	return len(m.Bookings), nil
}

// ── Conference Service Mock ───────────────────────────────────────

type MockConferenceService struct {
	Link string
	Err  error
}

func (m *MockConferenceService) CreateLink(_ context.Context, _ uuid.UUID) (string, error) {
	return m.Link, m.Err
}
