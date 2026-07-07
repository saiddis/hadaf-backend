// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package repositories

import "time"

/*
USERS
*/
type dbUser struct {
	ID                int        `db:"id"`
	OAuthProviderName *string    `db:"oauth_provider_name"`
	OAuthUserID       *string    `db:"oauth_user_id"`
	AvatarURL         *string    `db:"avatar_url"`
	InstitutionID     *int       `db:"institution_id"`
	FullName          *string    `db:"full_name"`
	Phone             *string    `db:"phone"`
	Email             *string    `db:"email"`
	Password          *string    `db:"password"`
	Role              string     `db:"role"`
	IsActive          bool       `db:"is_active"`
	IsApproved        bool       `db:"is_approved"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         *time.Time `db:"updated_at"`
	IsDeleted         bool       `db:"is_deleted"`
	DeletedAt         *time.Time `db:"deleted_at"`
}

/*
OTP
*/
type dbOtp struct {
	ID         int        `db:"id"`
	Attempt    int        `db:"attempt"`
	Receiver   string     `db:"receiver"`
	Method     *string    `db:"method"`
	OTPCode    string     `db:"otp_code"`
	IsVerified bool       `db:"is_verified"`
	SentAt     time.Time  `db:"sent_at"`
	ExpiresAt  *time.Time `db:"expires_at"`
	UpdatedAt  *time.Time `db:"updated_at"`
	IsDeleted  bool       `db:"is_deleted"`
	DeletedAt  *time.Time `db:"deleted_at"`
}

/*
INSTITUTIONS
*/
type dbInstitution struct {
	ID               int        `db:"id"`
	Name             string     `db:"name"`
	Type             *string    `db:"type"`
	City             *string    `db:"city"`
	Region           *string    `db:"region"`
	Address          *string    `db:"address"`
	Phone            *string    `db:"phone"`
	Email            *string    `db:"email"`
	Description      *string    `db:"description"`
	ActivityHours    *string    `db:"activity_hours"`
	Latitude         *float64   `db:"latitude"`
	Longitude        *float64   `db:"longitude"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        *time.Time `db:"updated_at"`
	IsDeleted        bool       `db:"is_deleted"`
	DeletedAt        *time.Time `db:"deleted_at"`
	WardsCount       int        `db:"wards_count"`
	ProhibitedItems  *string    `db:"prohibited_items"`
	RecommendedItems *string    `db:"recommended_items"`
}

/*
CATEGORIES
*/
type dbCategory struct {
	ID        int        `db:"id"`
	Name      string     `db:"name"`
	CreatedAt time.Time  `db:"created_at"`
	IsDeleted bool       `db:"is_deleted"`
	DeletedAt *time.Time `db:"deleted_at"`
}

/*
NEEDS
*/
type dbNeed struct {
	ID            int        `db:"id"`
	InstitutionID int        `db:"institution_id"`
	CategoryID    *int       `db:"category_id"`
	Name          string     `db:"name"`
	Description   *string    `db:"description"`
	Unit          string     `db:"unit"`
	RequiredQty   float64    `db:"required_qty"`
	ReceivedQty   float64    `db:"received_qty"`
	Urgency       string     `db:"urgency"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     *time.Time `db:"updated_at"`
	IsDeleted     bool       `db:"is_deleted"`
	DeletedAt     *time.Time `db:"deleted_at"`
}

/*
NEEDS HISTORY
*/
type dbNeedHistory struct {
	ID        int        `db:"id"`
	NeedID    int        `db:"need_id"`
	Comment   *string    `db:"comment"`
	CreatedAt time.Time  `db:"created_at"`
	IsDeleted bool       `db:"is_deleted"`
	DeletedAt *time.Time `db:"deleted_at"`
}

/*
BOOKINGS
*/
type dbBooking struct {
	ID        int        `db:"id"`
	UserID    int        `db:"user_id"`
	NeedID    int        `db:"need_id"`
	Quantity  float64    `db:"quantity"`
	Note      string     `db:"note"`
	Status    string     `db:"status"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt *time.Time `db:"updated_at"`
	IsDeleted bool       `db:"is_deleted"`
	DeletedAt *time.Time `db:"deleted_at"`
}
