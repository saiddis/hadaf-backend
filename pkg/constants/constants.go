// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package constants

type ctxKeyRequestID int

const RequestIDKey ctxKeyRequestID = 0

type ctxKey string

const CountryCodeKey ctxKey = "country_code"

const (
	RequestIDHeader = "X-Request-Id"
)

const (
	SendOTP = "send_otp"
)

const (
	AccessSubject  = "access"
	RefreshSubject = "refresh"
)

const (
	SSLModeDisable = "disable"
)

// Pagination defaults for list endpoints (GET /institutions, GET /events).
const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
)

// Application environments. These are the canonical values for APP_ENV; all
// environment checks must compare against these constants rather than ad-hoc
// string literals.
const (
	LocalAppEnv       = "local"
	DevelopmentAppEnv = "development"
	ProductionAppEnv  = "production"
)
