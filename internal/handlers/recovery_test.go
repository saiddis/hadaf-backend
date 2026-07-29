// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"shb/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestHandler_Recovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := zerolog.Nop()
	h := &Handler{metrics: metrics.New(), logger: &log}

	r := gin.New()
	r.Use(h.RequestID(), h.Recovery())
	r.GET("/boom", func(c *gin.Context) { panic("kaboom") })
	r.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	t.Run("panic recovered as 500", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("normal request unaffected", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "ok", w.Body.String())
	})
}
