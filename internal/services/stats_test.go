// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestService_GetPublicStats_Cases(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		stats := map[string]int{"volunteers": 10, "institutions": 3}
		d.Repo.On("GetPublicStats", ctx).Return(stats, nil)

		got, err := svc.GetPublicStats(ctx)
		require.NoError(t, err)
		require.Equal(t, stats, got)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetPublicStats", ctx).Return(nil, errors.New("db"))

		got, err := svc.GetPublicStats(ctx)
		require.Error(t, err)
		require.Nil(t, got)
	})
}
