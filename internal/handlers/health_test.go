// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	servicemock "shb/pkg/mocks/services"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_HealthEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := zerolog.Nop()

	newRouter := func(svc IService) *gin.Engine {
		h := &Handler{service: svc, logger: &log}
		r := gin.New()
		r.GET("/healthz", h.healthz)
		r.GET("/readyz", h.readyz)
		return r
	}

	t.Run("healthz is always ok", func(t *testing.T) {
		r := newRouter(servicemock.NewMockIService(t))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("readyz ok when dependencies healthy", func(t *testing.T) {
		svc := servicemock.NewMockIService(t)
		svc.On("HealthCheck", mock.Anything).Return(nil)
		r := newRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("readyz 503 when a dependency is down", func(t *testing.T) {
		svc := servicemock.NewMockIService(t)
		svc.On("HealthCheck", mock.Anything).Return(errors.New("db down"))
		r := newRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		require.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}
