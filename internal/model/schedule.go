package model

import (
	"database/sql/driver"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// DaysOfWeek stores API values 1–7 (1=Mon, 7=Sun) as a PostgreSQL integer[].
// Conversion to Go's time.Weekday: goWD = time.Weekday(apiDay % 7)
// (7 % 7 = 0 = Sunday, 1 % 7 = 1 = Monday, etc.)
type DaysOfWeek []int

func (d DaysOfWeek) Value() (driver.Value, error)       { return pq.Array([]int(d)).Value() }
func (d *DaysOfWeek) Scan(value interface{}) error      { return pq.Array((*[]int)(d)).Scan(value) }

type Schedule struct {
	ID         uuid.UUID  `db:"id"           json:"id"`
	RoomID     uuid.UUID  `db:"room_id"      json:"roomId"`
	DaysOfWeek DaysOfWeek `db:"days_of_week" json:"daysOfWeek"`
	StartTime  string     `db:"start_time"   json:"startTime"` // "HH:MM" or "HH:MM:SS" from DB
	EndTime    string     `db:"end_time"     json:"endTime"`
	CreatedAt  time.Time  `db:"created_at"   json:"createdAt"`
}
