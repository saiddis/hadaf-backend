// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package models

import "time"

const (
	BeneficiaryStatusPending  = "pending"
	BeneficiaryStatusVerified = "verified"
	BeneficiaryStatusRejected = "rejected"

	DiagnosisCerebralPalsy = "cerebral_palsy"
)

type Beneficiary struct {
	ID              int        `json:"id"`
	UserID          int        `json:"user_id"`
	FullName        string     `json:"full_name"`
	BirthDate       time.Time  `json:"birth_date"`
	Diagnosis       string     `json:"diagnosis"`
	City            *string    `json:"city"`
	Region          *string    `json:"region"`
	ContactPhone    *string    `json:"contact_phone"`
	Status          string     `json:"status"`
	RejectionReason *string    `json:"rejection_reason"`
	VerifiedBy      *int       `json:"verified_by"`
	VerifiedAt      *time.Time `json:"verified_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at"`
	IsDeleted       bool       `json:"-"`
}
