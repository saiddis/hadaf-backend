// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services_test

import (
	"context"
	"errors"
	"testing"

	"shb/pkg/external/sms/smsProvider"

	"github.com/stretchr/testify/require"
)

func TestService_CheckSMSBalance_Cases(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		res := &smsProvider.BalanceResult{Balance: "100.50", Timestamp: "2026-06-18"}
		d.SMS.On("CheckBalance", ctx).Return(res, nil)

		got, err := svc.CheckSMSBalance(ctx)
		require.NoError(t, err)
		require.Equal(t, res, got)
	})

	t.Run("adapter error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.SMS.On("CheckBalance", ctx).Return(nil, errors.New("provider down"))

		got, err := svc.CheckSMSBalance(ctx)
		require.Error(t, err)
		require.Nil(t, got)
	})
}
