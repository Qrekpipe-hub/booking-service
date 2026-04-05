package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Qrekpipe-hub/booking-service/internal/middleware"
	"github.com/Qrekpipe-hub/booking-service/internal/service"
)

type BookingHandler struct {
	bookingSvc *service.BookingService
}

func NewBookingHandler(bookingSvc *service.BookingService) *BookingHandler {
	return &BookingHandler{bookingSvc: bookingSvc}
}

// POST /bookings/create  (user only)
func (h *BookingHandler) Create(c *gin.Context) {
	var req struct {
		SlotID               string `json:"slotId"               binding:"required"`
		CreateConferenceLink bool   `json:"createConferenceLink"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	slotID, err := uuid.Parse(req.SlotID)
	if err != nil {
		badRequest(c, "slotId must be a valid UUID")
		return
	}

	userID := middleware.GetUserID(c)

	booking, err := h.bookingSvc.Create(c.Request.Context(), service.CreateBookingInput{
		SlotID:               slotID,
		UserID:               userID,
		CreateConferenceLink: req.CreateConferenceLink,
	})
	if err != nil {
		switch err {
		case service.ErrSlotNotFound:
			notFound(c, "SLOT_NOT_FOUND", "slot not found")
		case service.ErrSlotInPast:
			badRequest(c, "cannot book a slot in the past")
		case service.ErrSlotBooked:
			conflict(c, "SLOT_ALREADY_BOOKED", "slot is already booked")
		default:
			internalError(c)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"booking": booking})
}

// POST /bookings/:bookingId/cancel  (user only — own bookings)
func (h *BookingHandler) Cancel(c *gin.Context) {
	bookingID, err := uuid.Parse(c.Param("bookingId"))
	if err != nil {
		badRequest(c, "invalid bookingId")
		return
	}

	userID := middleware.GetUserID(c)

	booking, err := h.bookingSvc.Cancel(c.Request.Context(), bookingID, userID)
	if err != nil {
		switch err {
		case service.ErrBookingNotFound:
			notFound(c, "BOOKING_NOT_FOUND", "booking not found")
		case service.ErrForbidden:
			forbidden(c, "cannot cancel another user's booking")
		default:
			internalError(c)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"booking": booking})
}

// GET /bookings/my  (user only)
func (h *BookingHandler) ListMy(c *gin.Context) {
	userID := middleware.GetUserID(c)

	bookings, err := h.bookingSvc.GetMyBookings(c.Request.Context(), userID)
	if err != nil {
		internalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"bookings": bookings})
}

// GET /bookings/list  (admin only)
func (h *BookingHandler) ListAll(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 20)

	if page < 1 {
		badRequest(c, "page must be >= 1")
		return
	}
	if pageSize < 1 || pageSize > 100 {
		badRequest(c, "pageSize must be between 1 and 100")
		return
	}

	bookings, total, err := h.bookingSvc.ListAll(c.Request.Context(), page, pageSize)
	if err != nil {
		internalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bookings": bookings,
		"pagination": gin.H{
			"page":     page,
			"pageSize": pageSize,
			"total":    total,
		},
	})
}

func queryInt(c *gin.Context, key string, defaultVal int) int {
	s := c.Query(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
