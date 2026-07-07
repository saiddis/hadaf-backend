// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package models

import "time"

type BeneficiaryMedia struct {
	ID            int        `json:"id"`
	BeneficiaryID int        `json:"beneficiary_id"`
	FilePath      string     `json:"-"`
	FileHash      string     `json:"file_hash"`
	OriginalName  *string    `json:"original_name"`
	MimeType      *string    `json:"mime_type"`
	IsApproved    bool       `json:"is_approved"`
	ReviewedBy    *int       `json:"reviewed_by"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	UploadedAt    time.Time  `json:"uploaded_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
	IsDeleted     bool       `json:"-"`
}
