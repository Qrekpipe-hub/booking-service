package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/example/booking-service/internal/model"
	"github.com/example/booking-service/internal/repository"
)

const (
	slotDuration  = 30 * time.Minute
	generateDays  = 14 // rolling window ahead
)

// SlotGenerator generates and extends slot windows for all schedules.
type SlotGenerator struct {
	slots     repository.SlotRepository
	schedules repository.ScheduleRepository
}

func NewSlotGenerator(slots repository.SlotRepository, schedules repository.ScheduleRepository) *SlotGenerator {
	return &SlotGenerator{slots: slots, schedules: schedules}
}

// GenerateForSchedule creates slots for the next `generateDays` days for a single schedule.
// Called immediately after a schedule is created.
func (g *SlotGenerator) GenerateForSchedule(ctx context.Context, schedule *model.Schedule) error {
	from := time.Now().UTC().Truncate(24 * time.Hour)
	to := from.AddDate(0, 0, generateDays)
	return g.generateRange(ctx, schedule, from, to)
}

// ExtendAll extends slot windows for all schedules that need it.
// Should be called periodically (e.g. daily) by the background goroutine.
func (g *SlotGenerator) ExtendAll(ctx context.Context) {
	schedules, err := g.slots.ListSchedulesWithRooms(ctx)
	if err != nil {
		log.Printf("slot generator: list schedules: %v", err)
		return
	}

	targetEnd := time.Now().UTC().Truncate(24 * time.Hour).AddDate(0, 0, generateDays)

	for _, sched := range schedules {
		maxDate, err := g.slots.MaxSlotDate(ctx, sched.RoomID)
		if err != nil {
			log.Printf("slot generator: max slot date for room %s: %v", sched.RoomID, err)
			continue
		}

		var from time.Time
		if maxDate == nil {
			from = time.Now().UTC().Truncate(24 * time.Hour)
		} else {
			from = maxDate.UTC().Truncate(24*time.Hour).Add(24 * time.Hour) // day after last slot
		}

		if from.After(targetEnd) {
			continue // window already extends far enough
		}

		if err := g.generateRange(ctx, &sched, from, targetEnd); err != nil {
			log.Printf("slot generator: generate for room %s: %v", sched.RoomID, err)
		}
	}
}

// generateRange creates slots for [from, to) for the given schedule.
func (g *SlotGenerator) generateRange(ctx context.Context, schedule *model.Schedule, from, to time.Time) error {
	startH, startM, err := parseHHMM(schedule.StartTime)
	if err != nil {
		return fmt.Errorf("parse start_time %q: %w", schedule.StartTime, err)
	}
	endH, endM, err := parseHHMM(schedule.EndTime)
	if err != nil {
		return fmt.Errorf("parse end_time %q: %w", schedule.EndTime, err)
	}

	allowedWeekdays := make(map[time.Weekday]bool)
	for _, apiDay := range schedule.DaysOfWeek {
		// API: 1=Mon..7=Sun → Go: time.Weekday(apiDay % 7)
		allowedWeekdays[time.Weekday(apiDay%7)] = true
	}

	var slots []model.Slot
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		if !allowedWeekdays[d.Weekday()] {
			continue
		}
		dayStart := time.Date(d.Year(), d.Month(), d.Day(), startH, startM, 0, 0, time.UTC)
		dayEnd := time.Date(d.Year(), d.Month(), d.Day(), endH, endM, 0, 0, time.UTC)

		for cur := dayStart; cur.Add(slotDuration).Before(dayEnd) || cur.Add(slotDuration).Equal(dayEnd); cur = cur.Add(slotDuration) {
			slots = append(slots, model.Slot{
				ID:         uuid.New(),
				RoomID:     schedule.RoomID,
				ScheduleID: schedule.ID,
				StartAt:    cur,
				EndAt:      cur.Add(slotDuration),
			})
		}
	}

	if len(slots) == 0 {
		return nil
	}
	return g.slots.BulkInsert(ctx, slots)
}

// parseHHMM parses "HH:MM" or "HH:MM:SS" (PostgreSQL TIME returns with seconds).
func parseHHMM(s string) (int, int, error) {
	// Strip seconds part if present ("09:00:00" → "09:00")
	if len(s) > 5 && s[5] == ':' {
		s = s[:5]
	}
	// Trim spaces
	s = strings.TrimSpace(s)

	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return 0, 0, fmt.Errorf("invalid time format %q: %w", s, err)
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("time out of range: %d:%02d", h, m)
	}
	return h, m, nil
}
