// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package cache

import (
	"context"
	"time"
)

// ICache defines the cache contract (e.g., Redis).
type ICache interface {
	// Set stores a value under the given key with the specified TTL.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Get retrieves the string value stored under the given key.
	Get(ctx context.Context, key string) (string, error)

	// Delete removes the value stored under the given key.
	Delete(ctx context.Context, key string) error

	// Increment atomically increments the numeric value stored under the given key by 1.
	Increment(ctx context.Context, key string) error

	// Ping verifies connectivity to the cache (used by readiness checks).
	Ping(ctx context.Context) error
}
