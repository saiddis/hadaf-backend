// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package repositories

import (
	"context"
	"fmt"
	"shb/internal/models"
	"shb/internal/repositories/filters"
)

func (r *Repository) CreateNeed(ctx context.Context, n *models.Need) (int, error) {
	query := `
		INSERT INTO needs (institution_id, category_id, name, description, unit, required_qty, received_qty, urgency)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	var id int
	err := r.postgres.QueryRow(ctx, query,
		n.InstitutionID, n.CategoryID, n.Name, n.Description, n.Unit, n.RequiredQty, n.ReceivedQty, n.Urgency,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("create need: %w", err)
	}
	return id, nil
}

func (r *Repository) GetNeedByID(ctx context.Context, id int) (*models.Need, error) {
	query := `SELECT id, institution_id, name, unit, required_qty, received_qty, urgency FROM needs WHERE id = $1`
	var n models.Need
	err := r.postgres.QueryRow(ctx, query, id).Scan(
		&n.ID, &n.InstitutionID, &n.Name, &n.Unit, &n.RequiredQty, &n.ReceivedQty, &n.Urgency,
	)
	if err != nil {
		return nil, fmt.Errorf("get need: %w", err)
	}
	return &n, nil
}

func (r *Repository) UpdateNeed(ctx context.Context, n *models.Need) error {
	query := `
		UPDATE needs 
		SET name=$1, description=$2, unit=$3, required_qty=$4, received_qty=$5, urgency=$6, updated_at=NOW()
		WHERE id=$7 AND is_deleted = false
	`
	result, err := r.postgres.Exec(ctx, query,
		n.Name, n.Description, n.Unit, n.RequiredQty, n.ReceivedQty, n.Urgency, n.ID,
	)
	if err != nil {
		return fmt.Errorf("update need: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("need not found or deleted")
	}
	return nil
}

func (r *Repository) DeleteNeed(ctx context.Context, id int) error {
	query := `UPDATE needs SET is_deleted = true, deleted_at = NOW() WHERE id = $1`
	result, err := r.postgres.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete need: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("need not found")
	}
	return nil
}

func (r *Repository) CreateNeedHistory(ctx context.Context, history *models.NeedsHistory) error {
	query := `INSERT INTO needs_history (need_id, comment, created_at) VALUES ($1, $2, NOW())`
	_, err := r.postgres.Exec(ctx, query, history.NeedID, history.Comment)
	if err != nil {
		return fmt.Errorf("create history: %w", err)
	}
	return nil
}

func (r *Repository) GetNeedsByInstitution(ctx context.Context, filter filters.NeedsFilter, institutionID int) ([]*models.Need, error) {
	query := `
		SELECT n.id, n.institution_id, n.name, n.description, n.unit, n.required_qty, n.received_qty,
			COALESCE((SELECT SUM(b.quantity) FROM bookings b WHERE b.need_id = n.id AND b.status = 'pending' AND b.is_deleted = false), 0) AS booked_qty,
			n.urgency, n.created_at
		FROM needs n
	`
	filterQuery, args := filters.GetNeedsByInstitution(filter, institutionID)
	query += filterQuery

	// Log the query for debugging
	r.logger.Info().Str("query", query).Interface("args", args).Msg("Fetching needs DB")

	rows, err := r.postgres.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// IMPORTANT: Initialize as empty slice, not nil.
	// This ensures it serializes to [] in JSON instead of null.
	needs := make([]*models.Need, 0)

	for rows.Next() {
		var n models.Need
		if err := rows.Scan(
			&n.ID, &n.InstitutionID, &n.Name, &n.Description, &n.Unit, &n.RequiredQty, &n.ReceivedQty, &n.BookedQty, &n.Urgency, &n.CreatedAt,
		); err != nil {
			return nil, err
		}
		needs = append(needs, &n)
	}

	r.logger.Info().Int("count", len(needs)).Msg("Fetched needs count")
	return needs, nil
}
