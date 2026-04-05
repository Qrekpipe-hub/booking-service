package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/google/uuid"
)

// ConferenceService выдаёт ссылку на конференцию для брони.
type ConferenceService interface {
	CreateLink(ctx context.Context, bookingID uuid.UUID) (string, error)
}

// MockConferenceService — заглушка внешнего сервиса конференций.
// С вероятностью 10% возвращает ошибку; бронь при этом создаётся без ссылки.
type MockConferenceService struct{}

func NewMockConferenceService() ConferenceService {
	return &MockConferenceService{}
}

func (m *MockConferenceService) CreateLink(ctx context.Context, bookingID uuid.UUID) (string, error) {
	if rand.Float32() < 0.1 {
		return "", fmt.Errorf("conference service unavailable")
	}
	link := fmt.Sprintf("https://conf.internal/room/%s", bookingID)
	log.Printf("conference link created for booking %s", bookingID)
	return link, nil
}
