package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/Qrekpipe-hub/booking-service/internal/model"
)

type slotRepo struct{ db *sqlx.DB }

func NewSlotRepository(db *sqlx.DB) SlotRepository {
	return &slotRepo{db: db}
}

// BulkInsert inserts multiple slots efficiently, ignoring duplicates.
func (r *slotRepo) BulkInsert(ctx context.Context, slots []model.Slot) error {
	if len(slots) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO slots (id, room_id, schedule_id, start_at, end_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (room_id, start_at) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("prepare bulk insert: %w", err)
	}
	defer stmt.Close()

	for _, s := range slots {
		if _, err := stmt.ExecContext(ctx, s.ID, s.RoomID, s.ScheduleID, s.StartAt, s.EndAt, time.Now().UTC()); err != nil {
			return fmt.Errorf("insert slot: %w", err)
		}
	}

	return tx.Commit()
}

// GetAvailable returns free future slots for a room on a given UTC date.
// Uses idx_slots_room_start for fast lookup – this is the hot endpoint.
func (r *slotRepo) GetAvailable(ctx context.Context, roomID uuid.UUID, date time.Time) ([]model.Slot, error) {
	// date boundaries in UTC
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id, s.room_id, s.schedule_id, s.start_at, s.end_at
		FROM slots s
		WHERE s.room_id = $1
		  AND s.start_at >= $2
		  AND s.start_at <  $3
		  AND s.start_at >  NOW()
		  AND NOT EXISTS (
		      SELECT 1 FROM bookings b
		      WHERE b.slot_id = s.id AND b.status = 'active'
		  )
		ORDER BY s.start_at`,
		roomID, dayStart, dayEnd,
	)
	if err != nil {
		return nil, fmt.Errorf("get available slots: %w", err)
	}
	defer rows.Close()

	var slots []model.Slot
	for rows.Next() {
		var s model.Slot
		if err := rows.Scan(&s.ID, &s.RoomID, &s.ScheduleID, &s.StartAt, &s.EndAt); err != nil {
			return nil, fmt.Errorf("scan slot: %w", err)
		}
		slots = append(slots, s)
	}
	if slots == nil {
		slots = []model.Slot{}
	}
	return slots, rows.Err()
}

func (r *slotRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Slot, error) {
	var s model.Slot
	err := r.db.GetContext(ctx, &s, `SELECT * FROM slots WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get slot by id: %w", err)
	}
	return &s, nil
}

func (r *slotRepo) MaxSlotDate(ctx context.Context, roomID uuid.UUID) (*time.Time, error) {
	var t *time.Time
	err := r.db.QueryRowContext(ctx, `SELECT MAX(start_at) FROM slots WHERE room_id = $1`, roomID).Scan(&t)
	if err != nil {
		return nil, fmt.Errorf("max slot date: %w", err)
	}
	return t, nil
}

// ListSchedulesWithRooms returns all schedules for the background generator.
func (r *slotRepo) ListSchedulesWithRooms(ctx context.Context) ([]model.Schedule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, room_id, days_of_week, start_time::text, end_time::text, created_at
		FROM schedules`)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	var schedules []model.Schedule
	for rows.Next() {
		var s model.Schedule
		if err := rows.Scan(&s.ID, &s.RoomID, pq.Array(&s.DaysOfWeek), &s.StartTime, &s.EndTime, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}
