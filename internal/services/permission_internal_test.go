// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services

import (
	"context"
	"testing"
	"time"

	"shb/internal/configs"
	"shb/internal/models"
	cachemock "shb/pkg/mocks/cache"
	emailmock "shb/pkg/mocks/email"
	fsmock "shb/pkg/mocks/fs"
	repomock "shb/pkg/mocks/repository"
	smsmock "shb/pkg/mocks/sms"
	tokenmock "shb/pkg/mocks/tokens"
	"shb/pkg/myerrors"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testCfg() *configs.ServiceConfig {
	return &configs.ServiceConfig{
		Security: configs.SecurityConfig{
			OTPLength:       6,
			OTPDuration:     5 * time.Minute,
			RefreshTokenTTL: 720 * time.Hour,
		},
	}
}

func Test_checkPermission(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockIRepository(t)
	log := zerolog.Nop()
	s := NewService(
		testCfg(),
		&log,
		repo,
		repomock.NewMockCampaignRepository(t),
		repomock.NewMockLedgerRepository(t),
		cachemock.NewMockICache(t),
		smsmock.NewMockISmsAdapter(t),
		tokenmock.NewMockITokenIssuer(t),
		fsmock.NewMockStorage(t),
		emailmock.NewMockIEmailAdapter(t),
	)

	t.Run("no role in context", func(t *testing.T) {
		err := s.checkPermission(ctx, 1)
		require.Error(t, err)
		var u myerrors.UnauthorizedErr
		require.ErrorAs(t, err, &u)
	})

	t.Run("super admin ok", func(t *testing.T) {
		c := context.WithValue(ctx, "role", models.RoleSuperAdmin)
		require.NoError(t, s.checkPermission(c, 1))
	})

	t.Run("employee missing user id in context", func(t *testing.T) {
		c := context.WithValue(ctx, "role", models.RoleEmployee)
		err := s.checkPermission(c, 1)
		require.Error(t, err)
		var u myerrors.UnauthorizedErr
		require.ErrorAs(t, err, &u)
	})

	t.Run("employee wrong institution", func(t *testing.T) {
		c := context.WithValue(context.WithValue(ctx, "role", models.RoleEmployee), "userID", 5)
		inst := 10
		repo.On("GetUserByID", mock.Anything, 5).Return(&models.User{ID: 5, Role: models.RoleEmployee, InstitutionID: &inst}, nil).Once()
		err := s.checkPermission(c, 99)
		require.Error(t, err)
		var f myerrors.ForbiddenErr
		require.ErrorAs(t, err, &f)
	})

	t.Run("volunteer forbidden", func(t *testing.T) {
		c := context.WithValue(context.WithValue(ctx, "role", models.RoleVolunteer), "userID", 1)
		repo.On("GetUserByID", mock.Anything, 1).Return(&models.User{ID: 1, Role: models.RoleVolunteer}, nil).Once()
		err := s.checkPermission(c, 1)
		require.Error(t, err)
		var f myerrors.ForbiddenErr
		require.ErrorAs(t, err, &f)
	})

	t.Run("employee same institution", func(t *testing.T) {
		c := context.WithValue(context.WithValue(ctx, "role", models.RoleEmployee), "userID", 5)
		inst := 7
		repo.On("GetUserByID", mock.Anything, 5).Return(&models.User{ID: 5, Role: models.RoleEmployee, InstitutionID: &inst}, nil).Once()
		require.NoError(t, s.checkPermission(c, 7))
	})
}
