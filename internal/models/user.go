// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package models

import (
	"time"
)

// User roles
const (
	RoleSuperAdmin = "super_admin"
	RoleEmployee   = "employee"
	RoleVolunteer  = "volunteer"
)

type User struct {
	ID                int        `json:"id"`
	OAuthProviderName *string    `json:"oauth_provider_name"`
	OAuthUserID       *string    `json:"oauth_user_id"`
	AvatarURL         *string    `json:"avatar_url"`
	InstitutionID     *int       `json:"institution_id"`
	FullName          *string    `json:"full_name"`
	Phone             *string    `json:"phone"`
	Email             *string    `json:"email"`
	Password          *string    `json:"password"`
	Role              string     `json:"role"`
	IsActive          bool       `json:"is_active"`
	IsApproved        bool       `json:"is_approved"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         *time.Time `json:"updated_at"`
	IsDeleted         bool       `json:"is_deleted"`
	DeletedAt         *time.Time `json:"deleted_at"`
}

type OTP struct {
	ID         int        `json:"id"`
	Attempt    int        `json:"attempt"`
	Receiver   string     `json:"receiver"`
	Method     *string    `json:"method"`
	OTPCode    string     `json:"otp_code"`
	IsVerified bool       `json:"is_verified"`
	SentAt     time.Time  `json:"sent_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
	IsDeleted  bool       `json:"is_deleted"`
	DeletedAt  *time.Time `json:"deleted_at"`
}

// UpdateProfileRequest describes the incoming data for a partial profile update.
type UpdateProfileRequest struct {
	FullName *string `json:"full_name" binding:"omitempty,max=100"`
	Phone    *string `json:"phone" binding:"omitempty,max=20"`
}