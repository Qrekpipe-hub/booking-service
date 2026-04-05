package model

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type User struct {
	ID           uuid.UUID `db:"id"            json:"id"`
	Email        string    `db:"email"         json:"email"`
	PasswordHash *string   `db:"password_hash" json:"-"`
	Role         Role      `db:"role"          json:"role"`
	CreatedAt    time.Time `db:"created_at"    json:"createdAt"`
}
