// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"shb/internal/configs"
	"shb/internal/models"
	"shb/internal/services"
	"shb/pkg/middlewares"
	repomock "shb/pkg/mocks/repository"
	servicemock "shb/pkg/mocks/services"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type campaignTestLimiter struct{}

func (campaignTestLimiter) Allow(context.Context, string, int, int) (bool, error) { return true, nil }
func (campaignTestLimiter) ResetAttempts(context.Context, string) error           { return nil }

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func TestCampaignRoutesAreRegisteredWithExpectedProtection(t *testing.T) {
	router, _, _ := newCampaignRouter(t)
	routes := map[string]bool{}
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	assert.True(t, routes["GET /api/v1/campaigns/:id"])
	assert.True(t, routes["GET /api/v1/campaigns/:id/ledger"])
	assert.True(t, routes["POST /api/v1/campaigns/:id/donate"])
	assert.True(t, routes["GET /api/v1/donations/my"])
}

func TestCampaignDetailIsPublic(t *testing.T) {
	router, campaignRepo, _ := newCampaignRouter(t)
	campaignRepo.On("GetCampaignDetails", mock.Anything, 7).Return(&models.PublicCampaign{
		ID:         7,
		Title:      "Campaign",
		ClinicName: stringPtr("Clinic"),
	}, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/7", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	require.Equal(t, http.StatusOK, response.Code)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"clinic_name":"Clinic"`)
	assert.NotContains(t, string(body), "bank_account")
}

func TestCampaignListIsPublicAndForcesActiveStatus(t *testing.T) {
	router, campaignRepo, _ := newCampaignRouter(t)
	campaignRepo.On("GetAllCampaigns", mock.Anything, models.CampaignListQuery{
		Status: models.CampaignStatusActive,
		Limit:  20,
		Offset: 0,
	}).Return(&models.CampaignPage{Items: []*models.PublicCampaign{}}, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns?status=draft", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	assert.Equal(t, http.StatusOK, response.Code)
}

func TestDonationRequiresAuthentication(t *testing.T) {
	router, _, _ := newCampaignRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/7/donate", strings.NewReader(`{"amount":10}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestMyDonationsRequiresAuthentication(t *testing.T) {
	router, _, _ := newCampaignRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/donations/my", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestDonateUsesAuthenticatedUser(t *testing.T) {
	service := servicemock.NewMockIService(t)
	log := zerolog.Nop()
	handler := &Handler{service: service, logger: &log}

	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/7/donate", strings.NewReader(`{"amount":10}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "id", Value: "7"}}
	context.Set("userID", 12)

	service.On("ProcessDonation", mock.Anything, models.DonationRequest{
		CampaignID:  7,
		Amount:      10,
		DonorUserID: 12,
	}).Return(&models.LedgerEntry{
		ID:         9,
		CampaignID: intPtr(7),
		Amount:     10,
	}, nil).Once()

	handler.donate(context)

	assert.Equal(t, http.StatusOK, context.Writer.Status())
}

func newCampaignRouter(t *testing.T) (*gin.Engine, *repomock.MockCampaignRepository, *repomock.MockLedgerRepository) {
	t.Helper()

	log := zerolog.Nop()
	repo := repomock.NewMockIRepository(t)
	campaignRepo := repomock.NewMockCampaignRepository(t)
	ledgerRepo := repomock.NewMockLedgerRepository(t)
	service := services.NewService(
		&configs.ServiceConfig{},
		&log,
		repo,
		campaignRepo,
		ledgerRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	cfg := &configs.Config{}
	cfg.App.Env = "production"
	handler := NewHandler(
		service,
		campaignTestLimiter{},
		middlewares.NewMiddleware("campaign-test-secret"),
		&log,
		cfg,
	)

	return handler.InitRoutes(), campaignRepo, ledgerRepo
}
