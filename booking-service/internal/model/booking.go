package model

import (
	"time"

	"github.com/google/uuid"
)

type BookingStatus string

const (
	BookingStatusActive    BookingStatus = "active"
	BookingStatusCancelled BookingStatus = "cancelled"
)

type Booking struct {
	ID             uuid.UUID     `db:"id"              json:"id"`
	SlotID         uuid.UUID     `db:"slot_id"         json:"slotId"`
	UserID         uuid.UUID     `db:"user_id"         json:"userId"`
	Status         BookingStatus `db:"status"          json:"status"`
	ConferenceLink *string       `db:"conference_link" json:"conferenceLink,omitempty"`
	CreatedAt      time.Time     `db:"created_at"      json:"createdAt"`
}


