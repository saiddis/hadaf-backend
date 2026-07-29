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

func TestService_GetAllVacancies(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		vacancies := []*models.Vacancy{{ID: 1}, {ID: 2}}
		d.Repo.On("GetAllVacancies", ctx).Return(vacancies, nil)

		got, err := svc.GetAllVacancies(ctx)
		require.NoError(t, err)
		require.Equal(t, vacancies, got)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetAllVacancies", ctx).Return(nil, errors.New("db"))

		got, err := svc.GetAllVacancies(ctx)
		require.Error(t, err)
		require.Nil(t, got)
	})
}

func TestService_GetVacancyByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		vacancy := &models.Vacancy{ID: 7}
		d.Repo.On("GetVacancyByID", ctx, 7).Return(vacancy, nil)

		got, err := svc.GetVacancyByID(ctx, 7)
		require.NoError(t, err)
		require.Equal(t, vacancy, got)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetVacancyByID", ctx, 7).Return(nil, errors.New("db"))

		got, err := svc.GetVacancyByID(ctx, 7)
		require.Error(t, err)
		require.Nil(t, got)
	})
}
