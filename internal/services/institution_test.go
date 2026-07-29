// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services_test

import (
	"context"
	"errors"
	"testing"

	"shb/internal/models"

	"github.com/stretchr/testify/require"
)

func TestService_GetAllInstitutions(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		q := models.InstitutionListQuery{Limit: 10, Offset: 0}
		page := &models.InstitutionPage{Total: 2, Limit: 10}
		d.Repo.On("GetAllInstitutions", ctx, q).Return(page, nil)

		got, err := svc.GetAllInstitutions(ctx, q)
		require.NoError(t, err)
		require.Equal(t, page, got)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		q := models.InstitutionListQuery{Limit: 10}
		d.Repo.On("GetAllInstitutions", ctx, q).Return(nil, errors.New("db"))

		got, err := svc.GetAllInstitutions(ctx, q)
		require.Error(t, err)
		require.Nil(t, got)
	})
}

func TestService_CreateInstitution(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		inst := &models.Institution{Name: "Home"}
		d.Repo.On("CreateInstitution", ctx, inst).Return(11, nil)

		id, err := svc.CreateInstitution(ctx, inst)
		require.NoError(t, err)
		require.Equal(t, 11, id)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		inst := &models.Institution{Name: "Home"}
		d.Repo.On("CreateInstitution", ctx, inst).Return(0, errors.New("db"))

		id, err := svc.CreateInstitution(ctx, inst)
		require.Error(t, err)
		require.Equal(t, 0, id)
	})
}

func TestService_GetInstitutionByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		inst := &models.Institution{ID: 5, Name: "Home"}
		d.Repo.On("GetInstitutionByID", ctx, 5).Return(inst, nil)

		got, err := svc.GetInstitutionByID(ctx, 5)
		require.NoError(t, err)
		require.Equal(t, inst, got)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetInstitutionByID", ctx, 5).Return(nil, errors.New("db"))

		got, err := svc.GetInstitutionByID(ctx, 5)
		require.Error(t, err)
		require.Nil(t, got)
	})
}
