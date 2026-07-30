// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package models

import "time"

const (
	LedgerTypeDonation           = "donation"
	LedgerTypeOverflowToProvider = "overflow_out"
	LedgerTypeOverflowToGeneral  = "overflow_in"
	LedgerTypeRefund             = "refund"
	LedgerTypePaymentToProvider  = "payment_to_provider"
)

type LedgerEntry struct {
	ID           int       `json:"id"`
	CampaignID   *int      `json:"campaign_id"`
	DonorUserID  *int      `json:"-"`
	Type         string    `json:"type"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	PaymentRef   *string   `json:"-"`
	Description  *string   `json:"description"`
	DonorName    *string   `json:"donor_name"`
	IsAnonymous  bool      `json:"is_anonymous"`
	DonorMessage *string   `json:"donor_message"`
	CreatedAt    time.Time `json:"created_at"`
}

type LedgerEntryPage struct {
	Items  []*LedgerEntry `json:"items"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

type LedgerEntryListQuery struct {
	CampaignID    int
	DonorUserID   int
	Limit         int
	Offset        int
	MaskAnonymous bool
}
