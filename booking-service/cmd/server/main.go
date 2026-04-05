package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/booking-service/internal/config"
	"github.com/example/booking-service/internal/db"
	"github.com/example/booking-service/internal/handler"
	"github.com/example/booking-service/internal/repository"
	"github.com/example/booking-service/internal/service"
)

func main() {
	cfg := config.Load()

	// ── Database ──────────────────────────────────────────────────
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	log.Println("migrations applied")

	// ── Repositories ──────────────────────────────────────────────
	userRepo     := repository.NewUserRepository(database)
	roomRepo     := repository.NewRoomRepository(database)
	scheduleRepo := repository.NewScheduleRepository(database)
	slotRepo     := repository.NewSlotRepository(database)
	bookingRepo  := repository.NewBookingRepository(database)

	// ── Services ──────────────────────────────────────────────────
	conferenceSvc := service.NewMockConferenceService()
	slotGenerator := service.NewSlotGenerator(slotRepo, scheduleRepo)

	authSvc     := service.NewAuthService(userRepo, cfg.JWTSecret)
	roomSvc     := service.NewRoomService(roomRepo)
	scheduleSvc := service.NewScheduleService(scheduleRepo, roomRepo, slotGenerator)
	bookingSvc  := service.NewBookingService(bookingRepo, slotRepo, conferenceSvc)

	// ── Background: extend slot window daily ───────────────────────
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		// Run once at startup to fill any gaps
		ctx := context.Background()
		slotGenerator.ExtendAll(ctx)
		for range ticker.C {
			slotGenerator.ExtendAll(ctx)
		}
	}()

	// ── HTTP server ───────────────────────────────────────────────
	router := handler.NewRouter(handler.Services{
		Auth:     authSvc,
		Room:     roomSvc,
		Schedule: scheduleSvc,
		Booking:  bookingSvc,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped")
}
