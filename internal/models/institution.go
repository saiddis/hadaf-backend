// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package models

import "time"

type Institution struct {
	ID               int        `json:"id"`
	Name             string     `json:"name"`
	Type             *string    `json:"type"`
	City             *string    `json:"city"`
	Region           *string    `json:"region"`
	Address          *string    `json:"address"`
	Phone            *string    `json:"phone"`
	Email            *string    `json:"email"`
	Description      *string    `json:"description"`
	ActivityHours    *string    `json:"activity_hours"`
	Latitude         *float64   `json:"latitude"`
	Longitude        *float64   `json:"longitude"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at"`
	IsDeleted        bool       `json:"is_deleted"`
	DeletedAt        *time.Time `json:"deleted_at"`
	NeedsCount       int        `json:"needs_count"`
	WardsCount       int        `json:"wards_count"`
	ProhibitedItems  *string    `json:"prohibited_items"`
	RecommendedItems *string    `json:"recommended_items"`
}
