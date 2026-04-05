package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/google/uuid"
)

// ConferenceService generates a conference link for a booking.
// Real implementations would call an external video-conferencing API.
type ConferenceService interface {
	CreateLink(ctx context.Context, bookingID uuid.UUID) (string, error)
}

// MockConferenceService simulates an external conference service with occasional failures.
// Failure scenarios modelled:
//  1. Random 10% unavailability (network / service down).
//  2. Error is logged and the booking proceeds without a link (best-effort).
//
// Decision rationale (see README §Conference Link):
// The booking is the primary resource; the conference link is supplementary.
// Failing to obtain a link must never block or roll back a successful booking.
// The caller receives the booking with conferenceLink=nil if the service is unavailable.
type MockConferenceService struct{}

func NewMockConferenceService() ConferenceService {
	return &MockConferenceService{}
}

func (m *MockConferenceService) CreateLink(ctx context.Context, bookingID uuid.UUID) (string, error) {
	// Simulate occasional unavailability
	if rand.Float32() < 0.1 {
		return "", fmt.Errorf("conference service temporarily unavailable")
	}
	link := fmt.Sprintf("https://meet.example.com/booking/%s", bookingID)
	log.Printf("conference: created link %s for booking %s", link, bookingID)
	return link, nil
}
