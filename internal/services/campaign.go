// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services

import (
	"context"
	"errors"
	"fmt"

	"shb/internal/models"
	"shb/internal/repositories"
	"shb/pkg/myerrors"
)

func (s *Service) GetAllCampaigns(ctx context.Context, q models.CampaignListQuery) (*models.CampaignPage, error) {
	return s.repo.GetAllCampaigns(ctx, q)
}

func (s *Service) GetCampaignByID(ctx context.Context, id int) (*models.PublicCampaign, error) {
	if id <= 0 {
		return nil, myerrors.NewBadRequestErr("invalid campaign id")
	}

	campaign, err := s.repo.GetCampaignDetails(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrCampaignNotFound) {
			return nil, fmt.Errorf("%w: could not find campaign with id %d", myerrors.ErrNotFound, id)
		}
		return nil, err
	}

	return campaign, nil
}

func (s *Service) GetDonationsByCampaign(ctx context.Context, campaignID int) (*models.LedgerEntryPage, error) {
	if campaignID <= 0 {
		return nil, myerrors.NewBadRequestErr("invalid campaign id")
	}

	if _, err := s.repo.GetCampaignByID(ctx, campaignID); err != nil {
		if errors.Is(err, repositories.ErrCampaignNotFound) {
			return nil, fmt.Errorf("%w: could not find campaign with id %d", myerrors.ErrNotFound, campaignID)
		}
		return nil, err
	}

	page, err := s.repo.GetAllLedgerEntries(ctx, models.LedgerEntryListQuery{
		CampaignID:    campaignID,
		MaskAnonymous: true,
	})
	if err != nil {
		return nil, err
	}

	return page, nil
}

func (s *Service) GetDonationsByDonorUser(ctx context.Context, userID int) (*models.LedgerEntryPage, error) {
	if userID <= 0 {
		return nil, myerrors.NewUnauthorizedErr("user not authenticated")
	}

	page, err := s.repo.GetAllLedgerEntries(ctx, models.LedgerEntryListQuery{
		DonorUserID: userID,
	})
	if err != nil {
		return nil, err
	}

	return page, nil
}
