package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/example/booking-service/internal/model"
)

type roomRepo struct{ db *sqlx.DB }

func NewRoomRepository(db *sqlx.DB) RoomRepository {
	return &roomRepo{db: db}
}

func (r *roomRepo) Create(ctx context.Context, room *model.Room) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO rooms (id, name, description, capacity, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		room.ID, room.Name, room.Description, room.Capacity, room.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create room: %w", err)
	}
	return nil
}

func (r *roomRepo) List(ctx context.Context) ([]model.Room, error) {
	var rooms []model.Room
	if err := r.db.SelectContext(ctx, &rooms, `SELECT * FROM rooms ORDER BY created_at`); err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	return rooms, nil
}

func (r *roomRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Room, error) {
	var room model.Room
	err := r.db.GetContext(ctx, &room, `SELECT * FROM rooms WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get room by id: %w", err)
	}
	return &room, nil
}
