// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services_test

import (
	"context"
	"testing"
	"time"

	"shb/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetCampaignByIDReturnsPublicCampaign(t *testing.T) {
	service, deps := newTestService(t)
	ctx := context.Background()
	campaign := &models.PublicCampaign{
		ID:         7,
		Title:      "Campaign",
		ClinicName: stringPtr("Clinic"),
	}

	deps.CampaignRepo.On("GetCampaignDetails", ctx, 7).Return(campaign, nil).Once()

	got, err := service.GetCampaignByID(ctx, 7)

	require.NoError(t, err)
	assert.Equal(t, campaign, got)
}

func TestProcessDonationRecordsActiveCampaignDonation(t *testing.T) {
	service, deps := newTestService(t)
	ctx := context.Background()
	createdAt := time.Now()

	deps.CampaignRepo.On("GetByID", ctx, 7).Return(&models.MedicalCampaign{
		ID:       7,
		Currency: "TJS",
		Status:   models.CampaignStatusActive,
	}, nil).Once()
	deps.LedgerRepo.
		On("RecordDonation", ctx, mock.AnythingOfType("*models.LedgerEntry")).
		Run(func(args mock.Arguments) {
			entry := args.Get(1).(*models.LedgerEntry)
			entry.ID = 91
			entry.CreatedAt = createdAt
			require.NotNil(t, entry.CampaignID)
			require.NotNil(t, entry.DonorUserID)
			assert.Equal(t, 7, *entry.CampaignID)
			assert.Equal(t, 12, *entry.DonorUserID)
			assert.Equal(t, models.LedgerTypeDonation, entry.Type)
			assert.Equal(t, 25.5, entry.Amount)
			assert.Equal(t, "TJS", entry.Currency)
		}).
		Return(nil).
		Once()

	receipt, err := service.ProcessDonation(ctx, models.DonationRequest{
		CampaignID:  7,
		Amount:      25.5,
		DonorUserID: 12,
	})

	require.NoError(t, err)
	assert.Equal(t, &models.LedgerEntry{
		ID:          91,
		CampaignID:  intPtr(7),
		DonorUserID: intPtr(12),
		Type:        models.LedgerTypeDonation,
		Amount:      25.5,
		Currency:    "TJS",
		CreatedAt:   createdAt,
	}, receipt)
}

func TestProcessDonationRejectsInactiveCampaign(t *testing.T) {
	service, deps := newTestService(t)
	ctx := context.Background()

	deps.CampaignRepo.On("GetByID", ctx, 7).Return(&models.MedicalCampaign{
		ID:     7,
		Status: models.CampaignStatusCompleted,
	}, nil).Once()

	_, err := service.ProcessDonation(ctx, models.DonationRequest{
		CampaignID:  7,
		Amount:      25,
		DonorUserID: 12,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "campaign is not active")
}

func TestGetDonationsByCampaignChecksCampaignAndReturnsEntries(t *testing.T) {
	service, deps := newTestService(t)
	ctx := context.Background()
	entries := []*models.LedgerEntry{{ID: 3, CampaignID: intPtr(7), Amount: 10}}

	deps.CampaignRepo.On("GetByID", ctx, 7).Return(&models.MedicalCampaign{ID: 7}, nil).Once()
	deps.LedgerRepo.On("GetAllLedgerEntries", ctx, models.LedgerEntryListQuery{
		CampaignID:    7,
		MaskAnonymous: true,
	}).Return(&models.LedgerEntryPage{Items: entries}, nil).Once()

	got, err := service.GetDonationsByCampaign(ctx, 7)

	require.NoError(t, err)
	assert.Equal(t, entries, got.Items)
}

func TestGetDonationsByDonorUserScopesToAuthenticatedUser(t *testing.T) {
	service, deps := newTestService(t)
	ctx := context.Background()
	donations := []*models.LedgerEntry{{ID: 4, CampaignID: intPtr(7), Amount: 15}}

	deps.LedgerRepo.On("GetAllLedgerEntries", ctx, models.LedgerEntryListQuery{
		DonorUserID: 12,
	}).Return(&models.LedgerEntryPage{Items: donations}, nil).Once()

	got, err := service.GetDonationsByDonorUser(ctx, 12)

	require.NoError(t, err)
	assert.Equal(t, donations, got.Items)
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}
