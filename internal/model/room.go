package model

import (
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID          uuid.UUID `db:"id"          json:"id"`
	Name        string    `db:"name"        json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`
	Capacity    *int      `db:"capacity"    json:"capacity,omitempty"`
	CreatedAt   time.Time `db:"created_at"  json:"createdAt"`
}
