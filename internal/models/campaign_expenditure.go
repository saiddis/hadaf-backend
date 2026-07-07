// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package models

import "time"

type CampaignExpenditure struct {
	ID          int        `json:"id"`
	CampaignID  int        `json:"campaign_id"`
	ProviderID  *int       `json:"provider_id"`
	Amount      float64    `json:"amount"`
	Description string     `json:"description"`
	ReceiptPath *string    `json:"-"`
	InvoicePath *string    `json:"-"`
	PaidAt      *time.Time `json:"paid_at"`
	CreatedBy   *int       `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
	IsDeleted   bool       `json:"-"`
}
