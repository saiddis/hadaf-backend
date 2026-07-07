// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package models

import "time"

const (
	LedgerTypeDonation          = "donation"
	LedgerTypePaymentToProvider = "payment_to_provider"
	LedgerTypeOverflowToGeneral = "overflow_to_general"
	LedgerTypeRefund            = "refund"
)

type LedgerEntry struct {
	ID           int       `json:"id"`
	CampaignID   *int      `json:"campaign_id"`
	DonorUserID  *int      `json:"donor_user_id"`
	Type         string    `json:"type"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	PaymentRef   *string   `json:"payment_ref"`
	Description  *string   `json:"description"`
	IsAnonymous  bool      `json:"is_anonymous"`
	DonorMessage *string   `json:"donor_message"`
	CreatedAt    time.Time `json:"created_at"`
}
