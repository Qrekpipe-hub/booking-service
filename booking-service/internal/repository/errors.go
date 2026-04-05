package repository

import (
	"errors"
	"strings"
)

// Domain errors returned by repositories.
var ErrSlotAlreadyBooked = errors.New("slot already booked")

// isUniqueViolation detects PostgreSQL unique-constraint violations (code 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
