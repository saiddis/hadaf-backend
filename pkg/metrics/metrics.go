// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

// Package metrics provides Prometheus instrumentation for the HTTP layer.
//
// It exposes a self-contained registry (including the default Go runtime and
// process collectors), a Gin middleware that records request counts and
// latencies, and an HTTP handler that serves the metrics in the Prometheus
// exposition format.
package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// namespace prefixes every metric exported by this service.
	namespace = "shb"
	// endpoint is the route the metrics are served on; it is excluded from
	// instrumentation so scrapes do not pollute the request statistics.
	endpoint = "/metrics"
)

// Endpoint returns the route path the Prometheus metrics are served on.
func Endpoint() string { return endpoint }

// Metrics owns the Prometheus registry and the HTTP collectors. Keeping the
// registry on the struct (instead of the global default) makes the
// instrumentation self-contained and safe to instantiate in tests.
type Metrics struct {
	registry         *prometheus.Registry
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	externalDuration *prometheus.HistogramVec
	panicsTotal      prometheus.Counter
}

// New constructs a Metrics instance with a fresh registry, the standard Go and
// process collectors, and the HTTP request collectors registered.
func New() *Metrics {
	registry := prometheus.NewRegistry()

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests processed, partitioned by method, route and status code.",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency in seconds, partitioned by method, route and status code.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	externalDuration := newExternalDuration()

	panicsTotal := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_panics_total",
			Help:      "Total number of panics recovered by the HTTP recovery middleware.",
		},
	)

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		requestsTotal,
		requestDuration,
		externalDuration,
		panicsTotal,
	)

	return &Metrics{
		registry:         registry,
		requestsTotal:    requestsTotal,
		requestDuration:  requestDuration,
		externalDuration: externalDuration,
		panicsTotal:      panicsTotal,
	}
}

// IncPanic records a recovered panic. Safe to call on a nil receiver so callers
// need not guard when metrics are disabled.
func (m *Metrics) IncPanic() {
	if m == nil {
		return
	}
	m.panicsTotal.Inc()
}

// Middleware returns a Gin middleware that records the count and latency of
// every request once the downstream handlers have run.
//
// The route template (e.g. "/api/v1/needs/:id") is used as the path label
// rather than the raw URL, which keeps the metric cardinality bounded.
// Requests that do not match a registered route are grouped under "unmatched".
func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == endpoint {
			c.Next()
			return
		}

		start := time.Now()

		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())

		m.requestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		m.requestDuration.WithLabelValues(c.Request.Method, path, status).Observe(time.Since(start).Seconds())
	}
}

// Handler returns a Gin handler that serves the Prometheus exposition format
// for this instance's registry.
func (m *Metrics) Handler() gin.HandlerFunc {
	h := promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
