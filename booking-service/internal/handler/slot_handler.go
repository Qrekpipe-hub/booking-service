package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Qrekpipe-hub/booking-service/internal/service"
)

type SlotHandler struct {
	bookingSvc *service.BookingService
	roomSvc    *service.RoomService
}

func NewSlotHandler(bookingSvc *service.BookingService, roomSvc *service.RoomService) *SlotHandler {
	return &SlotHandler{bookingSvc: bookingSvc, roomSvc: roomSvc}
}

// GET /rooms/:roomId/slots/list
func (h *SlotHandler) ListAvailable(c *gin.Context) {
	roomID, err := uuid.Parse(c.Param("roomId"))
	if err != nil {
		badRequest(c, "invalid roomId")
		return
	}

	dateStr := c.Query("date")
	if dateStr == "" {
		badRequest(c, "date query parameter is required")
		return
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		badRequest(c, "date must be in YYYY-MM-DD format")
		return
	}

	// Verify room exists
	room, err := h.roomSvc.GetByID(c.Request.Context(), roomID)
	if err != nil {
		internalError(c)
		return
	}
	if room == nil {
		notFound(c, "ROOM_NOT_FOUND", "room not found")
		return
	}

	slots, err := h.bookingSvc.GetAvailableSlots(c.Request.Context(), roomID, date)
	if err != nil {
		internalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"slots": slots})
}
