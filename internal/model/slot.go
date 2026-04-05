package model

import (
	"time"

	"github.com/google/uuid"
)

type Slot struct {
	ID         uuid.UUID `db:"id"          json:"id"`
	RoomID     uuid.UUID `db:"room_id"     json:"roomId"`
	ScheduleID uuid.UUID `db:"schedule_id" json:"-"`
	StartAt    time.Time `db:"start_at"    json:"start"`
	EndAt      time.Time `db:"end_at"      json:"end"`
}
