// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"shb/internal/models"
	"shb/pkg/myerrors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// GetUserByPhone finds a user by phone.
func (r *Repository) GetUserByPhone(ctx context.Context, phone string) (*models.User, error) {
	query := `
		SELECT
			id,
			oauth_provider_name,
			oauth_user_id,
			avatar_url,
			institution_id,
			full_name,
			phone,
			email,
			password,
			role,
			is_active,
			is_approved,
			created_at,
			updated_at
		FROM users
		WHERE phone = $1
	`
	var u dbUser
	err := r.postgres.QueryRow(ctx, query, phone).Scan(
		&u.ID,
		&u.OAuthProviderName,
		&u.OAuthUserID,
		&u.AvatarURL,
		&u.InstitutionID,
		&u.FullName,
		&u.Phone,
		&u.Email,
		&u.Password,
		&u.Role,
		&u.IsActive,
		&u.IsApproved,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, myerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get user by phone query: %w", err)
	}

	return u.ToDomain(), nil
}

// GetUserByID finds a user by ID.
func (r *Repository) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	query := `
		SELECT
			id,
			oauth_provider_name,
			oauth_user_id,
			avatar_url,
			institution_id,
			full_name,
			phone,
			email,
			password,
			role,
			is_active,
			is_approved,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`
	var u dbUser
	err := r.postgres.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.OAuthProviderName,
		&u.OAuthUserID,
		&u.AvatarURL,
		&u.InstitutionID,
		&u.FullName,
		&u.Phone,
		&u.Email,
		&u.Password,
		&u.Role,
		&u.IsActive,
		&u.IsApproved,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, myerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get user by id query: %w", err)
	}

	return u.ToDomain(), nil
}

// CreateUser creates a new user.
func (r *Repository) CreateUser(ctx context.Context, u *models.User) error {
	query := `
		INSERT INTO users (
			oauth_provider_name,
			oauth_user_id,
			avatar_url,
			institution_id,
			full_name,
			phone,
			email,
			password,
			role,
			is_active,
			is_approved,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		RETURNING id, created_at, updated_at
	`
	err := r.postgres.QueryRow(
		ctx,
		query,
		u.OAuthProviderName,
		u.OAuthUserID,
		u.AvatarURL,
		u.InstitutionID,
		u.FullName,
		u.Phone,
		u.Email,
		u.Password,
		u.Role,
		u.IsActive,
		u.IsApproved,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create user query: %w", err)
	}
	return nil
}

func (r *Repository) ActivateUser(ctx context.Context, id int) error {
	query := `UPDATE users SET is_active = true, updated_at = NOW() WHERE id = $1`
	_, err := r.postgres.Exec(ctx, query, id)
	return err
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	// 1. Normalize email
	email = strings.ToLower(strings.TrimSpace(email))

	const query = `
		SELECT
			id,
			oauth_provider_name,
			oauth_user_id,
			avatar_url,
			institution_id,
			full_name,
			phone,
			email,
			password,
			role,
			is_active,
			is_approved,
			created_at
		FROM users
		WHERE email = $1
	`

	var u dbUser
	err := r.postgres.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.OAuthProviderName,
		&u.OAuthUserID,
		&u.AvatarURL,
		&u.InstitutionID,
		&u.FullName,
		&u.Phone,
		&u.Email,
		&u.Password,
		&u.Role,
		&u.IsActive,
		&u.IsApproved,
		&u.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, myerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return u.ToDomain(), nil
}

func (r *Repository) GetUserByOAuthInfo(ctx context.Context, oauthUserID, oauthProviderName string) (*models.User, error) {
	const query = `
		SELECT
			id,
			oauth_provider_name,
			oauth_user_id,
			avatar_url,
			institution_id,
			full_name,
			phone,
			email,
			password,
			role,
			is_active,
			is_approved,
			created_at,
			updated_at
		FROM users
		WHERE oauth_user_id = $1
			AND oauth_provider_name = $2
			AND deleted_at IS NULL
	`

	var u dbUser
	err := r.postgres.QueryRow(ctx, query, oauthUserID, oauthProviderName).Scan(
		&u.ID,
		&u.OAuthProviderName,
		&u.OAuthUserID,
		&u.AvatarURL,
		&u.InstitutionID,
		&u.FullName,
		&u.Phone,
		&u.Email,
		&u.Password,
		&u.Role,
		&u.IsActive,
		&u.IsApproved,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, myerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return u.ToDomain(), nil
}

// UpdateUserOAuthInfoByEmail sets oauth_user_id, oauth_provider_name and
// avatar_url using data from info.
func (r *Repository) UpdateUserOAuthInfoByEmail(ctx context.Context, info models.OAuthUserInfo) (*models.User, error) {
	const query = `
		UPDATE users SET
			oauth_user_id = $1,
			oauth_provider_name = $2,
			avatar_url = $3,
			updated_at = NOW()
		WHERE email = $4
		RETURNING
			id,
			oauth_provider_name,
			oauth_user_id,
			avatar_url,
			institution_id,
			full_name,
			phone,
			email,
			password,
			role,
			is_active,
			is_approved,
			created_at,
			updated_at
	`
	var u dbUser
	err := r.postgres.QueryRow(
		ctx,
		query,
		info.ID,
		info.OAuthProviderName,
		info.AvatarURL,
		info.Email,
	).Scan(
		&u.ID,
		&u.OAuthProviderName,
		&u.OAuthUserID,
		&u.AvatarURL,
		&u.InstitutionID,
		&u.FullName,
		&u.Phone,
		&u.Email,
		&u.Password,
		&u.Role,
		&u.IsActive,
		&u.IsApproved,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u.ToDomain(), nil
}

// UpdateProfile updates the full_name and phone fields of a user if provided.
func (r *Repository) UpdateProfile(ctx context.Context, id int, req models.UpdateProfileRequest) error {
	const query = `
		UPDATE users
		SET
			full_name = COALESCE($1, full_name),
			phone = COALESCE($2, phone),
			updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
	`

	res, err := r.postgres.Exec(ctx, query, req.FullName, req.Phone, id)
	if err != nil {
		return fmt.Errorf("update user profile query: %w", err)
	}

	if res.RowsAffected() == 0 {
    	return myerrors.ErrNotFound 
	}

	return nil
}