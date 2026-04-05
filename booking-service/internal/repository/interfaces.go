package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/example/booking-service/internal/model"
)

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
}

type RoomRepository interface {
	Create(ctx context.Context, room *model.Room) error
	List(ctx context.Context) ([]model.Room, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Room, error)
}

type ScheduleRepository interface {
	Create(ctx context.Context, schedule *model.Schedule) error
	GetByRoomID(ctx context.Context, roomID uuid.UUID) (*model.Schedule, error)
}

type SlotRepository interface {
	BulkInsert(ctx context.Context, slots []model.Slot) error
	GetAvailable(ctx context.Context, roomID uuid.UUID, date time.Time) ([]model.Slot, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Slot, error)
	MaxSlotDate(ctx context.Context, roomID uuid.UUID) (*time.Time, error)
	// ListSchedulesWithRooms returns all schedules for the background slot generator.
	ListSchedulesWithRooms(ctx context.Context) ([]model.Schedule, error)
}

type BookingRepository interface {
	Create(ctx context.Context, booking *model.Booking) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Booking, error)
	Cancel(ctx context.Context, id uuid.UUID) error
	// ListByUserFuture returns bookings for slots that start at or after `now`.
	ListByUserFuture(ctx context.Context, userID uuid.UUID, now time.Time) ([]model.Booking, error)
	ListAll(ctx context.Context, limit, offset int) ([]model.Booking, error)
	CountAll(ctx context.Context) (int, error)
}
