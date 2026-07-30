// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services_test

import (
	"testing"
	"time"

	"shb/internal/configs"
	"shb/internal/services"
	cachemock "shb/pkg/mocks/cache"
	emailmock "shb/pkg/mocks/email"
	fsmock "shb/pkg/mocks/fs"
	repomock "shb/pkg/mocks/repository"
	smsmock "shb/pkg/mocks/sms"
	tokenmock "shb/pkg/mocks/tokens"

	"github.com/rs/zerolog"
)

func testServiceConfig() *configs.ServiceConfig {
	return &configs.ServiceConfig{
		Security: configs.SecurityConfig{
			OTPLength:       6,
			OTPDuration:     5 * time.Minute,
			RefreshTokenTTL: 720 * time.Hour,
		},
	}
}

type testDeps struct {
	Repo         *repomock.MockIRepository
	CampaignRepo *repomock.MockCampaignRepository
	LedgerRepo   *repomock.MockLedgerRepository
	Cache        *cachemock.MockICache
	SMS          *smsmock.MockISmsAdapter
	Token        *tokenmock.MockITokenIssuer
	FS           *fsmock.MockStorage
	Email        *emailmock.MockIEmailAdapter
}

func newTestService(t *testing.T) (*services.Service, testDeps) {
	t.Helper()
	d := testDeps{
		Repo:         repomock.NewMockIRepository(t),
		CampaignRepo: repomock.NewMockCampaignRepository(t),
		LedgerRepo:   repomock.NewMockLedgerRepository(t),
		Cache:        cachemock.NewMockICache(t),
		SMS:          smsmock.NewMockISmsAdapter(t),
		Token:        tokenmock.NewMockITokenIssuer(t),
		FS:           fsmock.NewMockStorage(t),
		Email:        emailmock.NewMockIEmailAdapter(t),
	}
	log := zerolog.Nop()
	svc := services.NewService(
		testServiceConfig(),
		&log,
		d.Repo,
		d.Cache,
		d.SMS,
		d.Token,
		d.FS,
		d.Email,
	)
	return svc, d
}
