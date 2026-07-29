// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"shb/internal/models"
	"time"
)

func (r *Repository) CreateBooking(ctx context.Context, booking *models.Booking) (int, error) {
	query := `
		INSERT INTO bookings (user_id, need_id, quantity, note, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	var id int
	var createdAt time.Time
	err := r.postgres.QueryRow(ctx, query,
		booking.UserID, booking.NeedID, booking.Quantity, booking.Note, booking.Status,
	).Scan(&id, &createdAt)

	if err != nil {
		return 0, fmt.Errorf("create booking: %w", err)
	}
	booking.ID = id
	booking.CreatedAt = createdAt
	return id, nil
}

func (r *Repository) GetBookingByID(ctx context.Context, id int) (*models.Booking, error) {
	query := `
		SELECT id, user_id, need_id, quantity, note, status, created_at, updated_at, is_deleted, deleted_at
		FROM bookings
		WHERE id = $1 AND is_deleted = false
	`
	var b dbBooking
	err := r.postgres.QueryRow(ctx, query, id).Scan(
		&b.ID, &b.UserID, &b.NeedID, &b.Quantity, &b.Note, &b.Status,
		&b.CreatedAt, &b.UpdatedAt, &b.IsDeleted, &b.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("booking not found: %w", err)
		}
		return nil, fmt.Errorf("get booking by id: %w", err)
	}
	return b.ToDomain(), nil
}

func (r *Repository) GetActiveBookingByUserAndNeed(ctx context.Context, userID, needID int) (*models.Booking, error) {
	query := `
		SELECT id, user_id, need_id, quantity, note, status, created_at, updated_at, is_deleted, deleted_at
		FROM bookings
		WHERE user_id = $1 AND need_id = $2 
		  AND status NOT IN ('cancelled', 'rejected') 
		  AND is_deleted = false
		LIMIT 1
	`
	var b dbBooking
	err := r.postgres.QueryRow(ctx, query, userID, needID).Scan(
		&b.ID, &b.UserID, &b.NeedID, &b.Quantity, &b.Note, &b.Status,
		&b.CreatedAt, &b.UpdatedAt, &b.IsDeleted, &b.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No active booking found, return nil without error
		}
		return nil, fmt.Errorf("check active booking: %w", err)
	}
	return b.ToDomain(), nil
}

func (r *Repository) GetBookingsByNeed(ctx context.Context, needID int) ([]*models.Booking, error) {
	query := `
		SELECT id, user_id, need_id, quantity, note, status, created_at, updated_at, is_deleted, deleted_at
		FROM bookings
		WHERE need_id = $1 AND is_deleted = false
		ORDER BY created_at DESC
	`
	rows, err := r.postgres.Query(ctx, query, needID)
	if err != nil {
		return nil, fmt.Errorf("get bookings by need: %w", err)
	}
	defer rows.Close()

	var bookings []*models.Booking
	for rows.Next() {
		var b dbBooking
		if err := rows.Scan(
			&b.ID, &b.UserID, &b.NeedID, &b.Quantity, &b.Note, &b.Status,
			&b.CreatedAt, &b.UpdatedAt, &b.IsDeleted, &b.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		bookings = append(bookings, b.ToDomain())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return bookings, nil
}

func (r *Repository) GetBookingsByUser(ctx context.Context, userID int) ([]*models.Booking, error) {
	query := `
		SELECT b.id, b.user_id, b.need_id, b.quantity, b.note, b.status, b.created_at, b.updated_at, b.is_deleted, b.deleted_at,
		       n.name, i.name, i.id
		FROM bookings b
		JOIN needs n ON b.need_id = n.id
		JOIN institutions i ON n.institution_id = i.id
		WHERE b.user_id = $1 AND b.is_deleted = false
		ORDER BY b.created_at DESC
	`
	rows, err := r.postgres.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get bookings by user: %w", err)
	}
	defer rows.Close()

	var bookings []*models.Booking
	for rows.Next() {
		var b dbBooking
		var needName, instName string
		var instID int
		if err := rows.Scan(
			&b.ID, &b.UserID, &b.NeedID, &b.Quantity, &b.Note, &b.Status,
			&b.CreatedAt, &b.UpdatedAt, &b.IsDeleted, &b.DeletedAt,
			&needName, &instName, &instID,
		); err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		mb := b.ToDomain()
		mb.NeedName = needName
		mb.InstitutionName = instName
		mb.InstitutionID = instID

		// Map the note field directly to PlannedDate if it exists, since the frontend uses note for planned_date
		mb.Note = b.Note

		bookings = append(bookings, mb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return bookings, nil
}

func (r *Repository) UpdateBookingQuantity(ctx context.Context, bookingID int, qty float64) error {
	query := `
		UPDATE bookings 
		SET quantity = $1, updated_at = NOW()
		WHERE id = $2 AND is_deleted = false AND status = 'pending'
	`
	result, err := r.postgres.Exec(ctx, query, qty, bookingID)
	if err != nil {
		return fmt.Errorf("update booking qty: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("booking not found or not pending")
	}
	return nil
}

func (r *Repository) UpdateBookingStatus(ctx context.Context, bookingID int, status string) error {
	query := `
		UPDATE bookings 
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND is_deleted = false
	`
	result, err := r.postgres.Exec(ctx, query, status, bookingID)
	if err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("booking not found or already deleted")
	}
	return nil
}

func (r *Repository) GetBookingsByInstitution(ctx context.Context, institutionID int) ([]*models.Booking, error) {
	query := `
		SELECT b.id, b.user_id, b.need_id, b.quantity, b.note, b.status, b.created_at, b.updated_at, b.is_deleted, b.deleted_at
		FROM bookings b
		INNER JOIN needs n ON b.need_id = n.id
		WHERE n.institution_id = $1 AND b.is_deleted = false AND n.is_deleted = false
		ORDER BY b.created_at DESC
	`
	rows, err := r.postgres.Query(ctx, query, institutionID)
	if err != nil {
		return nil, fmt.Errorf("get bookings by institution: %w", err)
	}
	defer rows.Close()

	var bookings []*models.Booking
	for rows.Next() {
		var b dbBooking
		if err := rows.Scan(
			&b.ID, &b.UserID, &b.NeedID, &b.Quantity, &b.Note, &b.Status,
			&b.CreatedAt, &b.UpdatedAt, &b.IsDeleted, &b.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		bookings = append(bookings, b.ToDomain())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return bookings, nil
}

func (r *Repository) IncrementReceivedQty(ctx context.Context, needID int, qty float64) error {
	query := `
		UPDATE needs
		SET received_qty = received_qty + $1, updated_at = NOW()
		WHERE id = $2 AND is_deleted = false
	`
	result, err := r.postgres.Exec(ctx, query, qty, needID)
	if err != nil {
		return fmt.Errorf("increment received qty: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("need not found or already deleted")
	}
	return nil
}

// CompleteBookingTx atomically marks a booking as completed and increments the
// received quantity on its need. Both writes happen in a single transaction, so
// the booking can never be marked completed without the quantity being applied
// (and vice versa).
func (r *Repository) CompleteBookingTx(ctx context.Context, bookingID, needID int, qty float64) error {
	tx, err := r.postgres.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin complete booking tx: %w", err)
	}
	// Rollback is a no-op once the transaction has been committed.
	defer func() { _ = tx.Rollback(ctx) }()

	statusRes, err := tx.Exec(ctx, `
		UPDATE bookings
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND is_deleted = false
	`, models.BookingStatusCompleted, bookingID)
	if err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}
	if statusRes.RowsAffected() == 0 {
		return fmt.Errorf("booking not found or already deleted")
	}

	qtyRes, err := tx.Exec(ctx, `
		UPDATE needs
		SET received_qty = received_qty + $1, updated_at = NOW()
		WHERE id = $2 AND is_deleted = false
	`, qty, needID)
	if err != nil {
		return fmt.Errorf("increment received qty: %w", err)
	}
	if qtyRes.RowsAffected() == 0 {
		return fmt.Errorf("need not found or already deleted")
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit complete booking tx: %w", err)
	}
	return nil
}
