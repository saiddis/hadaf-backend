// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package models

import "time"

type ServiceProvider struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	LegalName    *string    `json:"legal_name"`
	BankName     *string    `json:"bank_name"`
	BankAccount  *string    `json:"bank_account"`
	ContactPhone *string    `json:"contact_phone"`
	ContactEmail *string    `json:"contact_email"`
	Address      *string    `json:"address"`
	IsVerified   bool       `json:"is_verified"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
	IsDeleted    bool       `json:"-"`
}
