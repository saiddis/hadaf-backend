// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package models

import "time"

const (
	VerificationDocumentStatusPending  = "pending"
	VerificationDocumentStatusApproved = "approved"
	VerificationDocumentStatusRejected = "rejected"
)

type VerificationDocument struct {
	ID            int        `json:"id"`
	BeneficiaryID *int       `json:"beneficiary_id"`
	CampaignID    *int       `json:"campaign_id"`
	DocumentType  string     `json:"document_type"`
	FilePath      string     `json:"-"`
	FileHash      string     `json:"file_hash"`
	FileSize      *int64     `json:"file_size"`
	OriginalName  *string    `json:"original_name"`
	MimeType      *string    `json:"mime_type"`
	Status        string     `json:"status"`
	RejectionNote *string    `json:"rejection_note"`
	ReviewedBy    *int       `json:"reviewed_by"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	UploadedAt    time.Time  `json:"uploaded_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
	IsDeleted     bool       `json:"-"`
}
