// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package handlers

import (
	"strconv"

	"shb/internal/models"
	"shb/pkg/myerrors"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func (h *Handler) getActiveCampaigns(c *gin.Context) {
	ctx := c.Request.Context()

	log := zerolog.Ctx(ctx).With().Str("handler", "getActiveCampaigns").Logger()
	ctx = log.WithContext(ctx)
	c.Request = c.Request.WithContext(ctx)

	limit, offset, err := parseLimitOffset(c)
	if err != nil {
		h.handleError(c, err)
		return
	}

	page, err := h.service.GetAllCampaigns(
		ctx,
		models.CampaignListQuery{
			Status: models.CampaignStatusActive,
			Limit:  limit,
			Offset: offset,
		},
	)

	if err != nil {
		h.handleError(c, err)
		return
	}

	log.Debug().Int("total", page.Total).Msg("campaigns fetched")
	h.success(c, page)
}

func (h *Handler) getCampaignByID(c *gin.Context) {
	campaignID, err := strconv.Atoi(c.Param("id"))
	if err != nil || campaignID <= 0 {
		h.handleError(c, myerrors.NewBadRequestErr("invalid campaign id"))
		return
	}

	ctx := c.Request.Context()
	log := zerolog.Ctx(ctx).With().Str("handler", "getCampaignByID").Int("campaign_id", campaignID).Logger()
	ctx = log.WithContext(ctx)
	c.Request = c.Request.WithContext(ctx)

	campaign, err := h.service.GetCampaignByID(ctx, campaignID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	log.Debug().Msg("campaign fetched")
	h.success(c, campaign)
}

func (h *Handler) getCampaignLedger(c *gin.Context) {
	campaignID, err := strconv.Atoi(c.Param("id"))
	if err != nil || campaignID <= 0 {
		h.handleError(c, myerrors.NewBadRequestErr("invalid campaign id"))
		return
	}

	ctx := c.Request.Context()
	log := zerolog.Ctx(ctx).With().Str("handler", "getCampaignLedger").Int("campaign_id", campaignID).Logger()
	ctx = log.WithContext(ctx)
	c.Request = c.Request.WithContext(ctx)

	page, err := h.service.GetDonationsByCampaign(ctx, campaignID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	log.Debug().Int("count", len(page.Items)).Msg("campaign ledger fetched")
	h.success(c, page)
}

func (h *Handler) donate(c *gin.Context) {
	campaignID, err := strconv.Atoi(c.Param("id"))
	if err != nil || campaignID <= 0 {
		h.handleError(c, myerrors.NewBadRequestErr("invalid campaign id"))
		return
	}

	userID, shouldReturn := h.mustGetUserID(c)
	if shouldReturn {
		return
	}

	var input struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		h.handleError(c, myerrors.NewBadRequestErr("invalid donation amount"))
		return
	}

	ctx := c.Request.Context()
	log := zerolog.Ctx(ctx).With().Str("handler", "donate").Int("campaign_id", campaignID).Int("user_id", userID).Logger()
	ctx = log.WithContext(ctx)
	c.Request = c.Request.WithContext(ctx)

	receipt, err := h.service.ProcessDonation(ctx, campaignID, input.Amount, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	log.Debug().Int("ledger_id", receipt.ID).Float64("amount", receipt.Amount).Msg("donation processed")
	h.success(c, receipt)
}

func (h *Handler) getMyDonations(c *gin.Context) {
	userID, shouldReturn := h.mustGetUserID(c)
	if shouldReturn {
		return
	}

	ctx := c.Request.Context()
	log := zerolog.Ctx(ctx).With().Str("handler", "getMyDonations").Int("user_id", userID).Logger()
	ctx = log.WithContext(ctx)
	c.Request = c.Request.WithContext(ctx)

	page, err := h.service.GetDonationsByDonorUser(ctx, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	log.Debug().Int("count", len(page.Items)).Msg("user donations fetched")
	h.success(c, page)
}
