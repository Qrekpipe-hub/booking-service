package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/Qrekpipe-hub/booking-service/internal/model"
)

type bookingRepo struct{ db *sqlx.DB }

func NewBookingRepository(db *sqlx.DB) BookingRepository {
	return &bookingRepo{db: db}
}

// Create inserts a new booking. Returns ErrSlotAlreadyBooked if the slot is taken.
func (r *bookingRepo) Create(ctx context.Context, b *model.Booking) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO bookings (id, user_id, slot_id, status, conference_link, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		b.ID, b.UserID, b.SlotID, b.Status, b.ConferenceLink, b.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrSlotAlreadyBooked
		}
		return fmt.Errorf("create booking: %w", err)
	}
	return nil
}

func (r *bookingRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Booking, error) {
	var b model.Booking
	err := r.db.GetContext(ctx, &b, `SELECT * FROM bookings WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get booking by id: %w", err)
	}
	return &b, nil
}

// Cancel sets status = 'cancelled' (idempotent — no error if already cancelled).
func (r *bookingRepo) Cancel(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE bookings SET status = 'cancelled' WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("cancel booking: %w", err)
	}
	return nil
}

// ListByUserFuture returns active+cancelled bookings whose slot hasn't started yet.
func (r *bookingRepo) ListByUserFuture(ctx context.Context, userID uuid.UUID, now time.Time) ([]model.Booking, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT b.id, b.slot_id, b.user_id, b.status, b.conference_link, b.created_at
		FROM bookings b
		JOIN slots s ON s.id = b.slot_id
		WHERE b.user_id = $1
		  AND s.start_at >= $2
		ORDER BY s.start_at`,
		userID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("list bookings by user: %w", err)
	}
	defer rows.Close()

	var bookings []model.Booking
	for rows.Next() {
		var b model.Booking
		if err := rows.Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		bookings = append(bookings, b)
	}
	if bookings == nil {
		bookings = []model.Booking{}
	}
	return bookings, rows.Err()
}

// ListAll returns all bookings with pagination.
func (r *bookingRepo) ListAll(ctx context.Context, limit, offset int) ([]model.Booking, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, slot_id, user_id, status, conference_link, created_at
		FROM bookings
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list all bookings: %w", err)
	}
	defer rows.Close()

	var bookings []model.Booking
	for rows.Next() {
		var b model.Booking
		if err := rows.Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		bookings = append(bookings, b)
	}
	if bookings == nil {
		bookings = []model.Booking{}
	}
	return bookings, rows.Err()
}

func (r *bookingRepo) CountAll(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bookings`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count bookings: %w", err)
	}
	return count, nil
}
