// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"shb/internal/models"
	"shb/pkg/myerrors"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestService_Login(t *testing.T) {
	ctx := context.Background()
	email := "vol@test.com"
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	require.NoError(t, err)
	pass := string(hash)

	tests := []struct {
		name    string
		setup   func(d testDeps)
		pass    string
		wantErr bool
		as      func(*testing.T, error)
	}{
		{
			name: "wrong password",
			setup: func(d testDeps) {
				u := &models.User{
					ID: 1, Email: &email, Password: &pass, IsActive: true, IsApproved: true,
					Role: models.RoleVolunteer,
				}
				d.Repo.On("GetUserByEmail", ctx, email).Return(u, nil)
			},
			pass:    "wrong",
			wantErr: true,
			as: func(t *testing.T, err error) {
				var u myerrors.UnauthorizedErr
				require.ErrorAs(t, err, &u)
			},
		},
		{
			name: "success",
			setup: func(d testDeps) {
				u := &models.User{
					ID: 1, Email: &email, Password: &pass, IsActive: true, IsApproved: true,
					Role: models.RoleVolunteer,
				}
				d.Repo.On("GetUserByEmail", ctx, email).Return(u, nil)
				d.Token.On("IssueTokens", ctx, 1, models.RoleVolunteer, true).Return("acc", "ref", nil)
				d.Repo.On("SaveRefreshToken", ctx, 1, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
			},
			pass:    "secret",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newTestService(t)
			tt.setup(d)
			tok, err := svc.Login(ctx, email, tt.pass)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, tok)
				if tt.as != nil {
					tt.as(t, err)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, "acc", tok.AccessToken)
		})
	}
}

func TestService_RefreshTokens(t *testing.T) {
	ctx := context.Background()
	svc, d := newTestService(t)
	rt := "refresh-raw"
	d.Token.On("VerifyToken", ctx, rt).Return(&models.CustomClaims{UserID: 1, Role: models.RoleVolunteer, IsApproved: true}, nil)
	d.Repo.On("GetRefreshToken", ctx, mock.AnythingOfType("string")).Return(&models.RefreshToken{
		UserID: 1, IsRevoked: false, ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	// RefreshTokens now re-validates the user's current state before rotating.
	d.Repo.On("GetUserByID", ctx, 1).Return(&models.User{
		ID: 1, Role: models.RoleVolunteer, IsActive: true, IsApproved: true,
	}, nil)
	d.Repo.On("RevokeRefreshToken", ctx, mock.AnythingOfType("string")).Return(nil)
	d.Token.On("IssueTokens", ctx, 1, models.RoleVolunteer, true).Return("na", "nr", nil)
	d.Repo.On("SaveRefreshToken", ctx, 1, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)

	tok, err := svc.RefreshTokens(ctx, rt)
	require.NoError(t, err)
	require.Equal(t, "na", tok.AccessToken)
}

func TestService_RefreshTokens_inactive_user(t *testing.T) {
	ctx := context.Background()
	svc, d := newTestService(t)
	rt := "refresh-raw"
	d.Token.On("VerifyToken", ctx, rt).Return(&models.CustomClaims{UserID: 1, Role: models.RoleVolunteer, IsApproved: true}, nil)
	d.Repo.On("GetRefreshToken", ctx, mock.AnythingOfType("string")).Return(&models.RefreshToken{
		UserID: 1, IsRevoked: false, ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	// A deactivated user must not be able to refresh; all their tokens are revoked.
	d.Repo.On("GetUserByID", ctx, 1).Return(&models.User{ID: 1, IsActive: false}, nil)
	d.Repo.On("RevokeAllUserRefreshTokens", ctx, 1).Return(nil)

	tok, err := svc.RefreshTokens(ctx, rt)
	require.Error(t, err)
	require.Nil(t, tok)
	var ua myerrors.UnauthorizedErr
	require.ErrorAs(t, err, &ua)
}

func TestService_Login_inactive_user(t *testing.T) {
	ctx := context.Background()
	email := "in@test.com"
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	pass := string(hash)
	u := &models.User{
		ID: 1, Email: &email, Password: &pass, IsActive: false, IsApproved: true, Role: models.RoleVolunteer,
	}
	svc, d := newTestService(t)
	d.Repo.On("GetUserByEmail", ctx, email).Return(u, nil)
	_, err := svc.Login(ctx, email, "secret")
	require.Error(t, err)
	var unauth myerrors.UnauthorizedErr
	require.ErrorAs(t, err, &unauth)
}

func TestService_Login_user_not_found(t *testing.T) {
	ctx := context.Background()
	svc, d := newTestService(t)
	d.Repo.On("GetUserByEmail", ctx, "x@y.z").Return(nil, myerrors.ErrNotFound)
	_, err := svc.Login(ctx, "x@y.z", "pw")
	require.Error(t, err)
	var u myerrors.UnauthorizedErr
	require.ErrorAs(t, err, &u)
}

func TestService_RefreshTokens_revoked(t *testing.T) {
	ctx := context.Background()
	svc, d := newTestService(t)
	rt := "r1"
	d.Token.On("VerifyToken", ctx, rt).Return(&models.CustomClaims{UserID: 1}, nil)
	d.Repo.On("GetRefreshToken", ctx, mock.AnythingOfType("string")).Return(&models.RefreshToken{
		UserID: 1, IsRevoked: true,
	}, nil)
	d.Repo.On("RevokeAllUserRefreshTokens", ctx, 1).Return(nil)
	_, err := svc.RefreshTokens(ctx, rt)
	require.Error(t, err)
	var fe myerrors.ForbiddenErr
	require.ErrorAs(t, err, &fe)
}

func TestService_RefreshTokens_refreshNotInStore(t *testing.T) {
	ctx := context.Background()
	svc, d := newTestService(t)
	rt := "tok"
	d.Token.On("VerifyToken", ctx, rt).Return(&models.CustomClaims{UserID: 1}, nil)
	d.Repo.On("GetRefreshToken", ctx, mock.AnythingOfType("string")).Return(nil, errors.New("missing"))
	_, err := svc.RefreshTokens(ctx, rt)
	require.Error(t, err)
	var u myerrors.UnauthorizedErr
	require.ErrorAs(t, err, &u)
}

func TestService_RefreshTokens_expired(t *testing.T) {
	ctx := context.Background()
	svc, d := newTestService(t)
	rt := "r2"
	d.Token.On("VerifyToken", ctx, rt).Return(&models.CustomClaims{UserID: 1}, nil)
	d.Repo.On("GetRefreshToken", ctx, mock.AnythingOfType("string")).Return(&models.RefreshToken{
		UserID: 1, IsRevoked: false, ExpiresAt: time.Now().Add(-time.Hour),
	}, nil)
	_, err := svc.RefreshTokens(ctx, rt)
	require.Error(t, err)
	var u myerrors.UnauthorizedErr
	require.ErrorAs(t, err, &u)
}

func TestService_RefreshTokens_invalid(t *testing.T) {
	ctx := context.Background()
	svc, d := newTestService(t)
	d.Token.On("VerifyToken", ctx, "bad").Return(nil, errors.New("bad sig"))
	_, err := svc.RefreshTokens(ctx, "bad")
	require.Error(t, err)
	var u myerrors.UnauthorizedErr
	require.ErrorAs(t, err, &u)
}
