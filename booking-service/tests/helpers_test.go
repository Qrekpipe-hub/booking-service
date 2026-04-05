package tests

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/example/booking-service/internal/model"
)

// ── In-memory User Repo ───────────────────────────────────────────

type fakeUserRepo struct {
	byEmail map[string]*model.User
	byID    map[uuid.UUID]*model.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byEmail: map[string]*model.User{}, byID: map[uuid.UUID]*model.User{}}
}

func (r *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	return r.byID[id], nil
}
func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	return r.byEmail[email], nil
}
func (r *fakeUserRepo) Create(_ context.Context, u *model.User) error {
	if _, ok := r.byEmail[u.Email]; ok {
		return fmt.Errorf("email taken")
	}
	r.byEmail[u.Email] = u
	r.byID[u.ID] = u
	return nil
}

// ── In-memory Room Repo ───────────────────────────────────────────

type fakeRoomRepo struct {
	rooms map[uuid.UUID]*model.Room
}

func newFakeRoomRepo() *fakeRoomRepo {
	return &fakeRoomRepo{rooms: map[uuid.UUID]*model.Room{}}
}

func (r *fakeRoomRepo) Create(_ context.Context, room *model.Room) error {
	r.rooms[room.ID] = room
	return nil
}
func (r *fakeRoomRepo) List(_ context.Context) ([]model.Room, error) {
	var out []model.Room
	for _, v := range r.rooms {
		out = append(out, *v)
	}
	return out, nil
}
func (r *fakeRoomRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Room, error) {
	return r.rooms[id], nil
}

// ── In-memory Schedule Repo ───────────────────────────────────────

type fakeScheduleRepo struct {
	byRoom map[uuid.UUID]*model.Schedule
}

func newFakeScheduleRepo() *fakeScheduleRepo {
	return &fakeScheduleRepo{byRoom: map[uuid.UUID]*model.Schedule{}}
}

func (r *fakeScheduleRepo) Create(_ context.Context, s *model.Schedule) error {
	r.byRoom[s.RoomID] = s
	return nil
}
func (r *fakeScheduleRepo) GetByRoomID(_ context.Context, roomID uuid.UUID) (*model.Schedule, error) {
	return r.byRoom[roomID], nil
}

// ── Shared fakeSlotRepo (wraps mocks.MockSlotRepo with ListSchedules) ─

type fakeSlotRepo struct {
	slots     map[uuid.UUID]*model.Slot
	schedRepo *fakeScheduleRepo
}

func newFakeSlotRepo(schedRepo *fakeScheduleRepo) *fakeSlotRepo {
	return &fakeSlotRepo{slots: map[uuid.UUID]*model.Slot{}, schedRepo: schedRepo}
}

func (r *fakeSlotRepo) BulkInsert(_ context.Context, slots []model.Slot) error {
	for i := range slots {
		s := slots[i]
		r.slots[s.ID] = &s
	}
	return nil
}

func (r *fakeSlotRepo) GetAvailable(_ context.Context, roomID uuid.UUID, date time.Time) ([]model.Slot, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	var result []model.Slot
	for _, s := range r.slots {
		if s.RoomID == roomID && !s.StartAt.Before(dayStart) && s.StartAt.Before(dayEnd) && s.StartAt.After(time.Now().UTC()) {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (r *fakeSlotRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Slot, error) {
	return r.slots[id], nil
}

func (r *fakeSlotRepo) MaxSlotDate(_ context.Context, roomID uuid.UUID) (*time.Time, error) {
	var max *time.Time
	for _, s := range r.slots {
		if s.RoomID == roomID {
			if max == nil || s.StartAt.After(*max) {
				t := s.StartAt
				max = &t
			}
		}
	}
	return max, nil
}

func (r *fakeSlotRepo) ListSchedulesWithRooms(_ context.Context) ([]model.Schedule, error) {
	var out []model.Schedule
	for _, s := range r.schedRepo.byRoom {
		out = append(out, *s)
	}
	return out, nil
}
