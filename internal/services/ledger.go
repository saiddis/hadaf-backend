package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"shb/internal/models"
	"shb/internal/repositories"
	"shb/pkg/myerrors"
	"time"
)

func (s *Service) ProcessDonation(ctx context.Context, campaignID int, amount float64, donorID *int) (*models.LedgerEntry, error) {
	if campaignID <= 0 {
		return nil, myerrors.NewBadRequestErr("campaign id required")
	}
	if donorID == nil || *donorID <= 0 {
		return nil, myerrors.NewUnauthorizedErr("user not authenticated")
	}
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return nil, myerrors.NewBadRequestErr("amount must be greater than 0")
	}

	campaign, err := s.repo.GetCampaignByID(ctx, campaignID)
	if err != nil {
		if errors.Is(err, repositories.ErrCampaignNotFound) {
			return nil, fmt.Errorf("%w: could not find campaign with id %d", myerrors.ErrNotFound, campaignID)
		}
		return nil, err
	}
	if campaign.Status != models.CampaignStatusActive {
		return nil, myerrors.NewBadRequestErr("campaign is not active")
	}

	entry := &models.LedgerEntry{
		CampaignID:  &campaignID,
		DonorUserID: donorID,
		Type:        models.LedgerTypeDonation,
		Amount:      amount,
		Currency:    campaign.Currency,
	}
	if err := s.repo.RecordDonation(ctx, entry); err != nil {
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

	updatedCampaign, err := s.repo.GetCampaignByID(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	overflow := updatedCampaign.CollectedAmount - updatedCampaign.TargetAmount

	if overflow >= 0 {
		if err = s.repo.UpdateCampaignStatus(ctx, updatedCampaign.ID, models.CampaignStatusReadyForPayment); err != nil {
			return nil, err
		}
	}

	if overflow > 0 {
		overflowToProvider := models.LedgerEntry{
			CampaignID:  &updatedCampaign.ID,
			Amount:      -overflow,
			Type:        models.LedgerTypeOverflowToProvider,
			DonorUserID: donorID,
			Currency:    updatedCampaign.Currency,
		}
		if err = s.repo.CreateLedgerEntry(ctx, &overflowToProvider); err != nil {
			return nil, err
		}

		overflowToGeneralFund := models.LedgerEntry{
			CampaignID:  nil,
			Amount:      overflow,
			Type:        models.LedgerTypeOverflowToGeneral,
			DonorUserID: donorID,
			Currency:    updatedCampaign.Currency,
		}
		if err = s.repo.CreateLedgerEntry(ctx, &overflowToGeneralFund); err != nil {
			return nil, err
		}
	}

	return entry, nil
}

func (s *Service) PayServiceProvider(ctx context.Context, campaignID int, amount float64, receiptPath string) (*models.CampaignExpenditure, error) {
	if campaignID <= 0 {
		return nil, myerrors.NewBadRequestErr("campaign id required")
	}
	if amount <= 0 {
		return nil, myerrors.NewBadRequestErr("invalid amount")
	}

	campaign, err := s.repo.GetCampaignByID(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving campaign by id: %w", err)
	}

	now := time.Now()
	expend := models.CampaignExpenditure{
		CampaignID: campaignID,
		Amount:     amount,
		PaidAt:     &now,
		CreatedAt:  now,
	}

	if receiptPath != "" {
		expend.ReceiptPath = &receiptPath
	}

	if err := s.repo.CreateCampaignExpenditure(ctx, &expend); err != nil {
		return nil, fmt.Errorf("error creating campaign expenditure: %v", err)
	}

	ledger := models.LedgerEntry{
		CampaignID:  &campaignID,
		Amount:      -amount,
		CreatedAt:   now,
		Currency:    campaign.Currency,
		Type:        models.LedgerTypePaymentToProvider,
		IsAnonymous: false, // TODO: true or false?
	}

	if err := s.repo.CreateLedgerEntry(ctx, &ledger); err != nil {
		return nil, fmt.Errorf("error creating ledger entry: %v", err)
	}

	// TODO: determinte why campaign status should become compeleted
	if err := s.repo.UpdateCampaignStatus(ctx, campaignID, models.CampaignStatusCompleted); err != nil {
		return nil, fmt.Errorf("error updating campaign status: %w", err)
	}

	return &expend, nil
}
