package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Qrekpipe-hub/booking-service/internal/model"
	"github.com/Qrekpipe-hub/booking-service/internal/repository"
)

var ErrScheduleExists = errors.New("schedule already exists for this room")
var ErrRoomNotFound = errors.New("room not found")

type ScheduleService struct {
	schedules repository.ScheduleRepository
	rooms     repository.RoomRepository
	generator *SlotGenerator
}

func NewScheduleService(
	schedules repository.ScheduleRepository,
	rooms repository.RoomRepository,
	generator *SlotGenerator,
) *ScheduleService {
	return &ScheduleService{schedules: schedules, rooms: rooms, generator: generator}
}

type CreateScheduleInput struct {
	RoomID     uuid.UUID
	DaysOfWeek []int
	StartTime  string // "HH:MM"
	EndTime    string // "HH:MM"
}

func (s *ScheduleService) Create(ctx context.Context, inp CreateScheduleInput) (*model.Schedule, error) {
	// Validate room exists
	room, err := s.rooms.GetByID(ctx, inp.RoomID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}

	// Validate daysOfWeek: 1–7
	if len(inp.DaysOfWeek) == 0 {
		return nil, fmt.Errorf("daysOfWeek must not be empty")
	}
	seen := make(map[int]bool)
	for _, d := range inp.DaysOfWeek {
		if d < 1 || d > 7 {
			return nil, fmt.Errorf("daysOfWeek values must be between 1 and 7, got %d", d)
		}
		seen[d] = true
	}

	// Validate time format and logic
	if err := validateTimeFormat(inp.StartTime); err != nil {
		return nil, fmt.Errorf("startTime: %w", err)
	}
	if err := validateTimeFormat(inp.EndTime); err != nil {
		return nil, fmt.Errorf("endTime: %w", err)
	}
	sh, sm, _ := parseHHMM(inp.StartTime)
	eh, em, _ := parseHHMM(inp.EndTime)
	startMinutes := sh*60 + sm
	endMinutes := eh*60 + em
	if endMinutes <= startMinutes {
		return nil, fmt.Errorf("endTime must be after startTime")
	}
	if (endMinutes-startMinutes) < 30 {
		return nil, fmt.Errorf("time range must be at least 30 minutes to produce a slot")
	}

	// Check uniqueness — one schedule per room
	existing, err := s.schedules.GetByRoomID(ctx, inp.RoomID)
	if err != nil {
		return nil, fmt.Errorf("check existing schedule: %w", err)
	}
	if existing != nil {
		return nil, ErrScheduleExists
	}

	schedule := &model.Schedule{
		ID:         uuid.New(),
		RoomID:     inp.RoomID,
		DaysOfWeek: model.DaysOfWeek(inp.DaysOfWeek),
		StartTime:  inp.StartTime,
		EndTime:    inp.EndTime,
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.schedules.Create(ctx, schedule); err != nil {
		return nil, fmt.Errorf("save schedule: %w", err)
	}

	// Generate slots synchronously — 14 days × ≤48 slots is fast enough.
	if genErr := s.generator.GenerateForSchedule(ctx, schedule); genErr != nil {
		log.Printf("schedule %s: slot generation error: %v", schedule.ID, genErr)
	}

	return schedule, nil
}

func (s *ScheduleService) GetByRoomID(ctx context.Context, roomID uuid.UUID) (*model.Schedule, error) {
	return s.schedules.GetByRoomID(ctx, roomID)
}

func validateTimeFormat(t string) error {
	t = strings.TrimSpace(t)
	var h, m int
	if _, err := fmt.Sscanf(t, "%d:%d", &h, &m); err != nil {
		return fmt.Errorf("invalid format %q, expected HH:MM", t)
	}
	if h < 0 || h > 23 {
		return fmt.Errorf("hour out of range: %d", h)
	}
	if m < 0 || m > 59 {
		return fmt.Errorf("minute out of range: %d", m)
	}
	return nil
}
