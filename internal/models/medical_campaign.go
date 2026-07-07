// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package models

import "time"

const (
	CampaignStatusDraft           = "draft"
	CampaignStatusPendingReview   = "pending_review"
	CampaignStatusActive          = "active"
	CampaignStatusReadyForPayment = "ready_for_payment"
	CampaignStatusCompleted       = "completed"
	CampaignStatusExpired         = "expired"
	CampaignStatusCancelled       = "cancelled"
)

type MedicalCampaign struct {
	ID              int        `json:"id"`
	BeneficiaryID   int        `json:"beneficiary_id"`
	ProviderID      *int       `json:"provider_id"`
	Title           string     `json:"title"`
	Description     *string    `json:"description"`
	TargetAmount    float64    `json:"target_amount"`
	CollectedAmount float64    `json:"collected_amount"`
	Currency        string     `json:"currency"`
	InvoiceNumber   *string    `json:"invoice_number"`
	Status          string     `json:"status"`
	Deadline        *time.Time `json:"deadline"`
	CompletedAt     *time.Time `json:"completed_at"`
	CancelledReason *string    `json:"cancelled_reason"`
	ReviewedBy      *int       `json:"reviewed_by"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at"`
	IsDeleted       bool       `json:"-"`
}

type CampaignListQuery struct {
	Status string `json:"status"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type CampaignPage struct {
	Items  []*MedicalCampaign `json:"items"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}
