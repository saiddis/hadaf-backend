// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services

import (
	"context"
	"errors"
	"fmt"
	"math"

	"shb/internal/models"
	"shb/internal/repositories"
	"shb/pkg/myerrors"
)

func (s *Service) GetAllCampaigns(ctx context.Context, q models.CampaignListQuery) (*models.CampaignPage, error) {
	return s.compaignRepo.GetAllCampaigns(ctx, q)
}

func (s *Service) GetCampaignByID(ctx context.Context, id int) (*models.PublicCampaign, error) {
	if id <= 0 {
		return nil, myerrors.NewBadRequestErr("invalid campaign id")
	}

	campaign, err := s.compaignRepo.GetCampaignDetails(ctx, id)
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

	if _, err := s.compaignRepo.GetByID(ctx, campaignID); err != nil {
		if errors.Is(err, repositories.ErrCampaignNotFound) {
			return nil, fmt.Errorf("%w: could not find campaign with id %d", myerrors.ErrNotFound, campaignID)
		}
		return nil, err
	}

	page, err := s.ledgerRepo.GetAllLedgerEntries(ctx, models.LedgerEntryListQuery{
		CampaignID:    campaignID,
		MaskAnonymous: true,
	})
	if err != nil {
		return nil, err
	}

	return page, nil
}

func (s *Service) ProcessDonation(ctx context.Context, donation models.DonationRequest) (*models.LedgerEntry, error) {
	if donation.CampaignID <= 0 {
		return nil, myerrors.NewBadRequestErr("invalid campaign id")
	}
	if donation.DonorUserID <= 0 {
		return nil, myerrors.NewUnauthorizedErr("user not authenticated")
	}
	if donation.Amount <= 0 || math.IsNaN(donation.Amount) || math.IsInf(donation.Amount, 0) {
		return nil, myerrors.NewBadRequestErr("amount must be greater than 0")
	}

	campaign, err := s.compaignRepo.GetByID(ctx, donation.CampaignID)
	if err != nil {
		if errors.Is(err, repositories.ErrCampaignNotFound) {
			return nil, fmt.Errorf("%w: could not find campaign with id %d", myerrors.ErrNotFound, donation.CampaignID)
		}
		return nil, err
	}
	if campaign.Status != models.CampaignStatusActive {
		return nil, myerrors.NewBadRequestErr("campaign is not active")
	}

	entry := &models.LedgerEntry{
		CampaignID:  &donation.CampaignID,
		DonorUserID: &donation.DonorUserID,
		Type:        models.LedgerTypeDonation,
		Amount:      donation.Amount,
		Currency:    campaign.Currency,
	}
	if err := s.ledgerRepo.RecordDonation(ctx, entry); err != nil {
		if err := errors.Unwrap(err); err != nil {
			switch err {
			case repositories.ErrInvalidAmount:
				return nil, myerrors.NewBadRequestErr("invalid amount")
			case repositories.ErrInvalidCampaignID:
				return nil, myerrors.NewBadRequestErr("invalid campaign id")
			}
		}
		return nil, err
	}

	return entry, nil
}

func (s *Service) GetDonationsByDonorUser(ctx context.Context, userID int) (*models.LedgerEntryPage, error) {
	if userID <= 0 {
		return nil, myerrors.NewUnauthorizedErr("user not authenticated")
	}

	page, err := s.ledgerRepo.GetAllLedgerEntries(ctx, models.LedgerEntryListQuery{
		DonorUserID: userID,
	})
	if err != nil {
		return nil, err
	}

	return page, nil
}
