// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package metrics

import (
	"context"
	"time"

	"shb/pkg/external/email"
	"shb/pkg/external/fs"
	"shb/pkg/external/sms"
	"shb/pkg/external/sms/smsProvider"

	"github.com/prometheus/client_golang/prometheus"
)

// newExternalDuration builds the histogram that tracks latency of outbound
// calls to third-party dependencies (SMS gateway, SMTP server, object storage).
// External integrations fail and slow down far more often than first-party
// code, so their latency and error rate are first-class signals.
func newExternalDuration() *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "external_request_duration_seconds",
			Help:      "Latency of outbound calls to external dependencies, partitioned by service, operation and status.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"service", "operation", "status"},
	)
}

// observeExternal records the duration and outcome of a single external call.
func (m *Metrics) observeExternal(service, operation string, start time.Time, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	m.externalDuration.WithLabelValues(service, operation, status).
		Observe(time.Since(start).Seconds())
}

// --- SMS ---

// InstrumentSMS wraps an SMS adapter so every call records latency/outcome.
// Returns the adapter unchanged on a nil receiver.
func (m *Metrics) InstrumentSMS(inner sms.ISmsAdapter) sms.ISmsAdapter {
	if m == nil {
		return inner
	}
	return &instrumentedSMS{inner: inner, m: m}
}

type instrumentedSMS struct {
	inner sms.ISmsAdapter
	m     *Metrics
}

func (s *instrumentedSMS) SendSms(ctx context.Context, phone, message, txnID string) error {
	start := time.Now()
	err := s.inner.SendSms(ctx, phone, message, txnID)
	s.m.observeExternal("sms", "send_sms", start, err)
	return err
}

func (s *instrumentedSMS) CheckBalance(ctx context.Context) (*smsProvider.BalanceResult, error) {
	start := time.Now()
	res, err := s.inner.CheckBalance(ctx)
	s.m.observeExternal("sms", "check_balance", start, err)
	return res, err
}

// --- Email ---

// InstrumentEmail wraps an email adapter so every call records latency/outcome.
func (m *Metrics) InstrumentEmail(inner email.IEmailAdapter) email.IEmailAdapter {
	if m == nil {
		return inner
	}
	return &instrumentedEmail{inner: inner, m: m}
}

type instrumentedEmail struct {
	inner email.IEmailAdapter
	m     *Metrics
}

func (e *instrumentedEmail) SendEmail(ctx context.Context, to, subject, body string) error {
	start := time.Now()
	err := e.inner.SendEmail(ctx, to, subject, body)
	e.m.observeExternal("email", "send_email", start, err)
	return err
}

// --- Object storage (MinIO) ---

// InstrumentStorage wraps a file-storage adapter so every call records
// latency/outcome.
func (m *Metrics) InstrumentStorage(inner fs.Storage) fs.Storage {
	if m == nil {
		return inner
	}
	return &instrumentedStorage{inner: inner, m: m}
}

type instrumentedStorage struct {
	inner fs.Storage
	m     *Metrics
}

func (s *instrumentedStorage) ReadFile(ctx context.Context, path string) (*fs.FileData, error) {
	start := time.Now()
	data, err := s.inner.ReadFile(ctx, path)
	s.m.observeExternal("storage", "read_file", start, err)
	return data, err
}

func (s *instrumentedStorage) WriteFile(ctx context.Context, path string, data *fs.FileData) (*fs.WriteResult, error) {
	start := time.Now()
	res, err := s.inner.WriteFile(ctx, path, data)
	s.m.observeExternal("storage", "write_file", start, err)
	return res, err
}
