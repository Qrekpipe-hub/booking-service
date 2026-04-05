package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Qrekpipe-hub/booking-service/internal/service"
)

type RoomHandler struct {
	rooms     *service.RoomService
	schedules *service.ScheduleService
}

func NewRoomHandler(rooms *service.RoomService, schedules *service.ScheduleService) *RoomHandler {
	return &RoomHandler{rooms: rooms, schedules: schedules}
}

// POST /rooms/create  (admin only)
func (h *RoomHandler) Create(c *gin.Context) {
	var req struct {
		Name        string  `json:"name"        binding:"required"`
		Description *string `json:"description"`
		Capacity    *int    `json:"capacity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	room, err := h.rooms.Create(c.Request.Context(), service.CreateRoomInput{
		Name:        req.Name,
		Description: req.Description,
		Capacity:    req.Capacity,
	})
	if err != nil {
		internalError(c)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"room": room})
}

// GET /rooms/list  (admin + user)
func (h *RoomHandler) List(c *gin.Context) {
	rooms, err := h.rooms.List(c.Request.Context())
	if err != nil {
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
}

// POST /rooms/:roomId/schedule/create  (admin only)
func (h *RoomHandler) CreateSchedule(c *gin.Context) {
	roomID, err := uuid.Parse(c.Param("roomId"))
	if err != nil {
		badRequest(c, "invalid roomId")
		return
	}

	var req struct {
		DaysOfWeek []int  `json:"daysOfWeek" binding:"required"`
		StartTime  string `json:"startTime"  binding:"required"`
		EndTime    string `json:"endTime"    binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	schedule, err := h.schedules.Create(c.Request.Context(), service.CreateScheduleInput{
		RoomID:     roomID,
		DaysOfWeek: req.DaysOfWeek,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
	})
	if err != nil {
		switch err {
		case service.ErrRoomNotFound:
			notFound(c, "ROOM_NOT_FOUND", "room not found")
		case service.ErrScheduleExists:
			conflict(c, "SCHEDULE_EXISTS", "schedule for this room already exists and cannot be changed")
		default:
			if isValidationErr(err) {
				badRequest(c, err.Error())
			} else {
				internalError(c)
			}
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"schedule": schedule})
}

// isValidationErr heuristic: service returns fmt.Errorf with user-visible messages for validation failures.
func isValidationErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, keyword := range []string{
		"daysOfWeek", "startTime", "endTime",
		"range", "format", "required", "empty", "after", "minute",
	} {
		if strings.Contains(msg, keyword) {
			return true
		}
	}
	return false
}
