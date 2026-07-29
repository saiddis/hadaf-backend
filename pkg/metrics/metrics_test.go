// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"shb/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTestRouter() (*gin.Engine, *metrics.Metrics) {
	gin.SetMode(gin.TestMode)
	m := metrics.New()

	r := gin.New()
	r.Use(m.Middleware())
	r.GET(metrics.Endpoint(), m.Handler())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	r.GET("/items/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r, m
}

func do(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	return w
}

func TestMetrics_ExposesRequestSeries(t *testing.T) {
	r, _ := newTestRouter()

	require.Equal(t, http.StatusOK, do(r, http.MethodGet, "/ping").Code)
	require.Equal(t, http.StatusNoContent, do(r, http.MethodGet, "/items/42").Code)

	body := do(r, http.MethodGet, metrics.Endpoint()).Body.String()

	require.Contains(t, body, "shb_http_requests_total")
	require.Contains(t, body, "shb_http_request_duration_seconds")
	require.Contains(t, body, `path="/ping"`)
	// Route template, not the raw URL, keeps cardinality bounded.
	require.Contains(t, body, `path="/items/:id"`)
	require.NotContains(t, body, `path="/items/42"`)
}

func TestMetrics_GroupsUnmatchedRoutes(t *testing.T) {
	r, _ := newTestRouter()

	require.Equal(t, http.StatusNotFound, do(r, http.MethodGet, "/does-not-exist").Code)

	body := do(r, http.MethodGet, metrics.Endpoint()).Body.String()
	require.Contains(t, body, `path="unmatched"`)
}

func TestMetrics_ExcludesScrapeEndpoint(t *testing.T) {
	r, _ := newTestRouter()

	// Scrape twice; the endpoint must not record a series about itself.
	do(r, http.MethodGet, metrics.Endpoint())
	body := do(r, http.MethodGet, metrics.Endpoint()).Body.String()

	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "shb_http_requests_total") {
			require.NotContains(t, line, metrics.Endpoint())
		}
	}
}
