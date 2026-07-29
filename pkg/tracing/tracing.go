// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

// Package tracing wires the OpenTelemetry tracer provider for the service.
//
// It is the code half of Phase 3 in docs/observability-report.md: the SDK that
// produces spans which otelgin / otelpgx / redisotel attach to, and the OTLP
// gRPC exporter that ships them to the OpenTelemetry Collector (and onward to
// Tempo).
//
// Design choices, on purpose:
//   - Tracing is OPT-IN. When disabled (or misconfigured) Init installs a no-op
//     tracer provider and returns a no-op shutdown, so the rest of the codebase
//     calls otel.Tracer(...) unconditionally and pays nothing when it is off.
//   - The W3C Trace Context + Baggage propagators are always set, even when the
//     exporter is off, so trace_id is still generated and can be correlated in
//     logs without standing up a Collector.
package tracing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config controls how the tracer provider is built. It is populated from the
// application config (which in turn reads OTEL_* environment variables).
type Config struct {
	// Enabled toggles span export. When false, Init only installs the
	// propagators and a no-op provider.
	Enabled bool
	// Endpoint is the OTLP/gRPC endpoint of the Collector, e.g.
	// "otel-collector:4317". Required when Enabled is true.
	Endpoint string
	// Insecure disables TLS on the OTLP connection. True for in-cluster /
	// docker-network transport; in production prefer mTLS (see report §2.5).
	Insecure bool
	// ServiceName is the logical service identity stamped on every span
	// (resource attribute service.name). Drives Tempo's service graph.
	ServiceName string
	// ServiceVersion is the build version, surfaced as service.version.
	ServiceVersion string
	// Environment is the deployment environment (deployment.environment).
	Environment string
	// SampleRatio is the head-based sampling probability in [0,1]. 1 traces
	// everything (fine for dev); lower it in production to control cost.
	SampleRatio float64
}

// noopShutdown is returned whenever there is nothing to flush, so callers can
// always `defer shutdown(ctx)` without a nil check.
func noopShutdown(context.Context) error { return nil }

// normalizeEndpoint coerces an endpoint into the "host:port" form that
// otlptracegrpc.WithEndpoint expects. It accepts the standard
// OTEL_EXPORTER_OTLP_ENDPOINT URL form too (http://host:port) by stripping the
// scheme and any trailing slash, so a scheme'd value does not silently fail.
func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return strings.TrimSuffix(endpoint, "/")
}

// Init configures the global OpenTelemetry tracer provider and text-map
// propagator and returns a shutdown function that flushes pending spans.
//
// The returned shutdown is always safe to call (never nil), even on error or
// when tracing is disabled.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	// Propagation works regardless of export: it lets us read/write the W3C
	// traceparent header and keep a trace_id for log correlation.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if !cfg.Enabled {
		// Leave the global provider as the SDK default no-op. No exporter, no
		// background goroutines, nothing to shut down.
		return noopShutdown, nil
	}

	if cfg.Endpoint == "" {
		return noopShutdown, fmt.Errorf("tracing enabled but OTEL exporter endpoint is empty")
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return noopShutdown, fmt.Errorf("build otel resource: %w", err)
	}

	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(normalizeEndpoint(cfg.Endpoint))}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	// A short dial timeout keeps app startup from hanging if the Collector is
	// down; the exporter retries in the background once the app is running.
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(dialCtx, opts...)
	if err != nil {
		return noopShutdown, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		// ParentBased keeps a trace consistent end-to-end: if an upstream
		// already sampled it, we honour that; otherwise sample at SampleRatio.
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
	)

	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
