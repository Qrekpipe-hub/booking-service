package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/example/booking-service/internal/model"
)

type scheduleRepo struct{ db *sqlx.DB }

func NewScheduleRepository(db *sqlx.DB) ScheduleRepository {
	return &scheduleRepo{db: db}
}

func (r *scheduleRepo) Create(ctx context.Context, s *model.Schedule) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO schedules (id, room_id, days_of_week, start_time, end_time, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		s.ID, s.RoomID, pq.Array([]int(s.DaysOfWeek)), s.StartTime, s.EndTime, s.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create schedule: %w", err)
	}
	return nil
}

func (r *scheduleRepo) GetByRoomID(ctx context.Context, roomID uuid.UUID) (*model.Schedule, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, room_id, days_of_week, start_time::text, end_time::text, created_at
		 FROM schedules WHERE room_id = $1`, roomID)

	var s model.Schedule
	err := row.Scan(&s.ID, &s.RoomID, pq.Array(&s.DaysOfWeek), &s.StartTime, &s.EndTime, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get schedule by room_id: %w", err)
	}
	return &s, nil
}
