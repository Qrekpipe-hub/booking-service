package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/booking-service/internal/middleware"
	"github.com/example/booking-service/internal/model"
	"github.com/example/booking-service/internal/service"
)

type Services struct {
	Auth     *service.AuthService
	Room     *service.RoomService
	Schedule *service.ScheduleService
	Booking  *service.BookingService
}

func NewRouter(svc Services) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// ── Health / info ──────────────────────────────────────────────
	r.GET("/_info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ── Handlers ───────────────────────────────────────────────────
	authH    := NewAuthHandler(svc.Auth)
	roomH    := NewRoomHandler(svc.Room, svc.Schedule)
	slotH    := NewSlotHandler(svc.Booking, svc.Room)
	bookingH := NewBookingHandler(svc.Booking)

	// ── Auth (public) ──────────────────────────────────────────────
	r.POST("/dummyLogin", authH.DummyLogin)
	r.POST("/register",   authH.Register)  // optional feature
	r.POST("/login",      authH.Login)     // optional feature

	// ── Authenticated routes ───────────────────────────────────────
	authMW := middleware.Auth(svc.Auth)

	// Rooms — list: admin + user; create: admin only
	r.GET("/rooms/list",
		authMW,
		roomH.List,
	)
	r.POST("/rooms/create",
		authMW,
		middleware.RequireRole(model.RoleAdmin),
		roomH.Create,
	)

	// Schedules — admin only
	r.POST("/rooms/:roomId/schedule/create",
		authMW,
		middleware.RequireRole(model.RoleAdmin),
		roomH.CreateSchedule,
	)

	// Slots — admin + user
	r.GET("/rooms/:roomId/slots/list",
		authMW,
		slotH.ListAvailable,
	)

	// Bookings — create/cancel/my: user only; list: admin only
	r.POST("/bookings/create",
		authMW,
		middleware.RequireRole(model.RoleUser),
		bookingH.Create,
	)
	r.GET("/bookings/my",
		authMW,
		middleware.RequireRole(model.RoleUser),
		bookingH.ListMy,
	)
	r.GET("/bookings/list",
		authMW,
		middleware.RequireRole(model.RoleAdmin),
		bookingH.ListAll,
	)
	r.POST("/bookings/:bookingId/cancel",
		authMW,
		middleware.RequireRole(model.RoleUser),
		bookingH.Cancel,
	)

	return r
}
