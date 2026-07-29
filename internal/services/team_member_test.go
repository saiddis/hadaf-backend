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

func TestService_GetAllTeamMembers(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		members := []*models.TeamMember{{ID: 1}, {ID: 2}}
		d.Repo.On("GetAllTeamMembers", ctx).Return(members, nil)

		got, err := svc.GetAllTeamMembers(ctx)
		require.NoError(t, err)
		require.Equal(t, members, got)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetAllTeamMembers", ctx).Return(nil, errors.New("db"))

		got, err := svc.GetAllTeamMembers(ctx)
		require.Error(t, err)
		require.Nil(t, got)
	})
}

func TestService_GetTeamMemberByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		member := &models.TeamMember{ID: 3}
		d.Repo.On("GetTeamMemberByID", ctx, 3).Return(member, nil)

		got, err := svc.GetTeamMemberByID(ctx, 3)
		require.NoError(t, err)
		require.Equal(t, member, got)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetTeamMemberByID", ctx, 3).Return(nil, errors.New("db"))

		got, err := svc.GetTeamMemberByID(ctx, 3)
		require.Error(t, err)
		require.Nil(t, got)
	})
}
