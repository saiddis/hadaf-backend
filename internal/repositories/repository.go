// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package repositories

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Repository struct {
	postgres *pgxpool.Pool
	logger   *zerolog.Logger
}

func NewRepository(postgresConn *pgxpool.Pool, log *zerolog.Logger) *Repository {
	return &Repository{postgres: postgresConn, logger: log}
}

// Ping verifies the database connection is alive. Used by readiness probes.
func (r *Repository) Ping(ctx context.Context) error {
	return r.postgres.Ping(ctx)
}

// PoolStats exposes the underlying pgx pool statistics for metrics collection.
func (r *Repository) PoolStats() *pgxpool.Stat {
	return r.postgres.Stat()
}
