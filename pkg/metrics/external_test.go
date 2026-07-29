// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package metrics_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"shb/pkg/external/fs"
	"shb/pkg/external/sms/smsProvider"
	"shb/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeSMS struct{ err error }

func (f fakeSMS) SendSms(context.Context, string, string, string) error { return f.err }
func (f fakeSMS) CheckBalance(context.Context) (*smsProvider.BalanceResult, error) {
	return &smsProvider.BalanceResult{}, f.err
}

type fakeEmail struct{ err error }

func (f fakeEmail) SendEmail(context.Context, string, string, string) error { return f.err }

type fakeStorage struct{ err error }

func (f fakeStorage) ReadFile(context.Context, string) (*fs.FileData, error) {
	return &fs.FileData{}, f.err
}
func (f fakeStorage) WriteFile(context.Context, string, *fs.FileData) (*fs.WriteResult, error) {
	return &fs.WriteResult{}, f.err
}

func TestMetrics_InstrumentsExternalCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := metrics.New()

	smsAdapter := m.InstrumentSMS(fakeSMS{})
	emailAdapter := m.InstrumentEmail(fakeEmail{})
	storageAdapter := m.InstrumentStorage(fakeStorage{err: errors.New("boom")})

	ctx := context.Background()
	require.NoError(t, smsAdapter.SendSms(ctx, "+992", "code", "1"))
	require.NoError(t, emailAdapter.SendEmail(ctx, "a@b.c", "s", "b"))
	_, _ = storageAdapter.WriteFile(ctx, "p", &fs.FileData{}) // returns error → status="error"

	r := gin.New()
	r.GET(metrics.Endpoint(), m.Handler())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, metrics.Endpoint(), nil))
	body := w.Body.String()

	require.Contains(t, body, "shb_external_request_duration_seconds")
	require.Contains(t, body, `service="sms"`)
	require.Contains(t, body, `operation="send_sms"`)
	require.Contains(t, body, `service="email"`)
	require.Contains(t, body, `service="storage"`)
	require.Contains(t, body, `status="error"`)
}
