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

func TestGenerateForSchedule_BasicSlotCount(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	scheduleRepo := &mockScheduleRepo{}
	gen := service.NewSlotGenerator(slotRepo, scheduleRepo)

	// Monday-only schedule, 09:00–11:00 → 4 slots per occurrence
	schedule := &model.Schedule{
		ID:         uuid.New(),
		RoomID:     uuid.New(),
		DaysOfWeek: model.DaysOfWeek{1}, // Monday (API encoding)
		StartTime:  "09:00",
		EndTime:    "11:00",
		CreatedAt:  time.Now().UTC(),
	}

	err := gen.GenerateForSchedule(context.Background(), schedule)
	require.NoError(t, err)

	// There should be slots; each Monday in window produces 4 (09:00, 09:30, 10:00, 10:30)
	assert.NotEmpty(t, slotRepo.Slots)
	for _, s := range slotRepo.Slots {
		assert.Equal(t, schedule.RoomID, s.RoomID)
		assert.Equal(t, 30*time.Minute, s.EndAt.Sub(s.StartAt), "slot must be exactly 30 minutes")
	}
}

func TestGenerateForSchedule_AllSlotsInCorrectWeekday(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	gen := service.NewSlotGenerator(slotRepo, &mockScheduleRepo{})

	// Wednesday only (API day = 3)
	schedule := &model.Schedule{
		ID:         uuid.New(),
		RoomID:     uuid.New(),
		DaysOfWeek: model.DaysOfWeek{3}, // Wednesday
		StartTime:  "10:00",
		EndTime:    "12:00",
	}

	err := gen.GenerateForSchedule(context.Background(), schedule)
	require.NoError(t, err)

	for _, s := range slotRepo.Slots {
		// API day 3 → Go weekday 3 = Wednesday
		assert.Equal(t, time.Wednesday, s.StartAt.UTC().Weekday(),
			"all slots must be on Wednesday")
	}
}

func TestGenerateForSchedule_SundayMapping(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	gen := service.NewSlotGenerator(slotRepo, &mockScheduleRepo{})

	// Sunday = API day 7 → Go weekday 0
	schedule := &model.Schedule{
		ID:         uuid.New(),
		RoomID:     uuid.New(),
		DaysOfWeek: model.DaysOfWeek{7},
		StartTime:  "08:00",
		EndTime:    "09:00",
	}

	err := gen.GenerateForSchedule(context.Background(), schedule)
	require.NoError(t, err)

	for _, s := range slotRepo.Slots {
		assert.Equal(t, time.Sunday, s.StartAt.UTC().Weekday(), "slots must be on Sunday")
	}
}

func TestGenerateForSchedule_NoSlotsIfEndBeforeStart(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	gen := service.NewSlotGenerator(slotRepo, &mockScheduleRepo{})

	// endTime == startTime → no slots
	schedule := &model.Schedule{
		ID:         uuid.New(),
		RoomID:     uuid.New(),
		DaysOfWeek: model.DaysOfWeek{1, 2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "09:00",
	}

	err := gen.GenerateForSchedule(context.Background(), schedule)
	require.NoError(t, err)
	assert.Empty(t, slotRepo.Slots, "no slots should be generated when end == start")
}

func TestGenerateForSchedule_SlotTimesAreUTC(t *testing.T) {
	slotRepo := mocks.NewMockSlotRepo()
	gen := service.NewSlotGenerator(slotRepo, &mockScheduleRepo{})

	schedule := &model.Schedule{
		ID:         uuid.New(),
		RoomID:     uuid.New(),
		DaysOfWeek: model.DaysOfWeek{1, 2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "10:00",
	}

	err := gen.GenerateForSchedule(context.Background(), schedule)
	require.NoError(t, err)

	for _, s := range slotRepo.Slots {
		assert.Equal(t, time.UTC, s.StartAt.Location(), "StartAt must be UTC")
		assert.Equal(t, time.UTC, s.EndAt.Location(), "EndAt must be UTC")
	}
}


func TestScheduleService_InvalidDayOfWeek(t *testing.T) {
	roomRepo := &mockRoomRepo{room: &model.Room{ID: uuid.New(), Name: "Test"}}
	scheduleRepo := &mockScheduleRepo{}
	gen := service.NewSlotGenerator(mocks.NewMockSlotRepo(), scheduleRepo)
	svc := service.NewScheduleService(scheduleRepo, roomRepo, gen)

	// Day 8 is invalid (valid range 1–7)
	_, err := svc.Create(context.Background(), service.CreateScheduleInput{
		RoomID:     roomRepo.room.ID,
		DaysOfWeek: []int{1, 8},
		StartTime:  "09:00",
		EndTime:    "18:00",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "daysOfWeek")
}

func TestScheduleService_DuplicateSchedule(t *testing.T) {
	roomRepo := &mockRoomRepo{room: &model.Room{ID: uuid.New(), Name: "Test"}}
	existing := &model.Schedule{ID: uuid.New(), RoomID: roomRepo.room.ID}
	scheduleRepo := &mockScheduleRepo{existing: existing}
	gen := service.NewSlotGenerator(mocks.NewMockSlotRepo(), scheduleRepo)
	svc := service.NewScheduleService(scheduleRepo, roomRepo, gen)

	_, err := svc.Create(context.Background(), service.CreateScheduleInput{
		RoomID:     roomRepo.room.ID,
		DaysOfWeek: []int{1},
		StartTime:  "09:00",
		EndTime:    "18:00",
	})
	assert.ErrorIs(t, err, service.ErrScheduleExists)
}

func TestScheduleService_RoomNotFound(t *testing.T) {
	roomRepo := &mockRoomRepo{room: nil}
	scheduleRepo := &mockScheduleRepo{}
	gen := service.NewSlotGenerator(mocks.NewMockSlotRepo(), scheduleRepo)
	svc := service.NewScheduleService(scheduleRepo, roomRepo, gen)

	_, err := svc.Create(context.Background(), service.CreateScheduleInput{
		RoomID:     uuid.New(),
		DaysOfWeek: []int{1},
		StartTime:  "09:00",
		EndTime:    "18:00",
	})
	assert.ErrorIs(t, err, service.ErrRoomNotFound)
}


type mockScheduleRepo struct {
	existing  *model.Schedule
	created   *model.Schedule
}

func (m *mockScheduleRepo) Create(_ context.Context, s *model.Schedule) error {
	m.created = s
	return nil
}

func (m *mockScheduleRepo) GetByRoomID(_ context.Context, _ uuid.UUID) (*model.Schedule, error) {
	return m.existing, nil
}

type mockRoomRepo struct {
	room *model.Room
}

func (m *mockRoomRepo) Create(_ context.Context, _ *model.Room) error { return nil }
func (m *mockRoomRepo) List(_ context.Context) ([]model.Room, error) {
	if m.room == nil {
		return nil, nil
	}
	return []model.Room{*m.room}, nil
}
func (m *mockRoomRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.Room, error) {
	return m.room, nil
}
