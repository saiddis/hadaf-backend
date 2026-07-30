// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package repositories

import (
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Repository struct {
	*CampaignLedgerRepository
	postgres *pgxpool.Pool
	logger   *zerolog.Logger
}

func NewRepository(postgresConn *pgxpool.Pool, log *zerolog.Logger) *Repository {
	return &Repository{
		CampaignLedgerRepository: NewCampaignLedgerRepository(postgresConn, slog.Default()),
		postgres:                 postgresConn,
		logger:                   log,
	}
}

func formatLimitOffset(limit, offset int) string {
	switch {
	case limit > 0 && offset > 0:
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	case limit > 0:
		return fmt.Sprintf("LIMIT %d", limit)
	case offset > 0:
		return fmt.Sprintf("OFFSET %d", offset)
	}
	return ""
}
