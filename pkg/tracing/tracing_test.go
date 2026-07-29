// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// When tracing is disabled, Init must still install the propagator (so trace_id
// correlation works without a Collector) and return a non-nil no-op shutdown.
func TestInit_DisabledInstallsPropagatorAndNoopShutdown(t *testing.T) {
	shutdown, err := Init(context.Background(), Config{Enabled: false})

	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))

	// The composite propagator must carry the W3C traceparent field.
	fields := otel.GetTextMapPropagator().Fields()
	assert.Contains(t, fields, "traceparent")
}

// Enabling tracing without an endpoint is a configuration error, but it must
// degrade gracefully: a usable no-op shutdown, never nil.
func TestInit_EnabledWithoutEndpointErrorsGracefully(t *testing.T) {
	shutdown, err := Init(context.Background(), Config{Enabled: true, Endpoint: ""})

	require.Error(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

// normalizeEndpoint must accept both bare host:port and the standard URL form
// so OTEL_EXPORTER_OTLP_ENDPOINT does not silently fail when given a scheme.
func TestNormalizeEndpoint(t *testing.T) {
	cases := map[string]string{
		"otel-collector:4317":         "otel-collector:4317",
		"http://otel-collector:4317":  "otel-collector:4317",
		"https://otel-collector:4317": "otel-collector:4317",
		"http://otel-collector:4317/": "otel-collector:4317",
		"localhost:4317":              "localhost:4317",
	}
	for in, want := range cases {
		assert.Equal(t, want, normalizeEndpoint(in), "input %q", in)
	}
}
