package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Qrekpipe-hub/booking-service/internal/model"
	"github.com/Qrekpipe-hub/booking-service/internal/repository"
)

type RoomService struct {
	rooms repository.RoomRepository
}

func NewRoomService(rooms repository.RoomRepository) *RoomService {
	return &RoomService{rooms: rooms}
}

type CreateRoomInput struct {
	Name        string
	Description *string
	Capacity    *int
}

func (s *RoomService) Create(ctx context.Context, inp CreateRoomInput) (*model.Room, error) {
	if inp.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	room := &model.Room{
		ID:          uuid.New(),
		Name:        inp.Name,
		Description: inp.Description,
		Capacity:    inp.Capacity,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.rooms.Create(ctx, room); err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}
	return room, nil
}

func (s *RoomService) List(ctx context.Context) ([]model.Room, error) {
	rooms, err := s.rooms.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	return rooms, nil
}

func (s *RoomService) GetByID(ctx context.Context, id uuid.UUID) (*model.Room, error) {
	room, err := s.rooms.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	return room, nil
}
