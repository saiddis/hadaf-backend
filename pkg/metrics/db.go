// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// RegisterDBPool exposes pgx connection-pool statistics as Prometheus gauges.
// The pool is sampled lazily on each scrape via GaugeFunc, so there is no
// background goroutine and the values are always current.
//
// Safe to call on a nil receiver or with a nil pool (no-op) so wiring code need
// not guard.
func (m *Metrics) RegisterDBPool(pool *pgxpool.Pool) {
	if m == nil || pool == nil {
		return
	}

	gauges := []struct {
		name string
		help string
		val  func() float64
	}{
		{"db_pool_total_conns", "Total number of connections currently in the pool.",
			func() float64 { return float64(pool.Stat().TotalConns()) }},
		{"db_pool_acquired_conns", "Number of currently acquired (in-use) connections.",
			func() float64 { return float64(pool.Stat().AcquiredConns()) }},
		{"db_pool_idle_conns", "Number of currently idle connections.",
			func() float64 { return float64(pool.Stat().IdleConns()) }},
		{"db_pool_max_conns", "Maximum size of the pool.",
			func() float64 { return float64(pool.Stat().MaxConns()) }},
	}

	for _, g := range gauges {
		m.registry.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{Namespace: namespace, Name: g.name, Help: g.help},
			g.val,
		))
	}
}
