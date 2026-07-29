// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestService_HealthCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("all dependencies healthy", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("Ping", ctx).Return(nil)
		d.Cache.On("Ping", ctx).Return(nil)
		require.NoError(t, svc.HealthCheck(ctx))
	})

	t.Run("database down", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("Ping", ctx).Return(errors.New("conn refused"))
		require.Error(t, svc.HealthCheck(ctx))
	})

	t.Run("cache down", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("Ping", ctx).Return(nil)
		d.Cache.On("Ping", ctx).Return(errors.New("redis down"))
		require.Error(t, svc.HealthCheck(ctx))
	})
}
