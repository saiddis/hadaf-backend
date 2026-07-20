// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"shb/internal/configs"
	"shb/internal/handlers"
	"shb/internal/repositories"
	"shb/internal/services"
	"shb/pkg/external/sms/smsProvider"
	"shb/pkg/middlewares"
	"shb/pkg/tokens/jwtToken"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

type allowAllLimiter struct{}

func (allowAllLimiter) Allow(context.Context, string, int, int) (bool, error) { return true, nil }
func (allowAllLimiter) ResetAttempts(context.Context, string) error           { return nil }

type smsNoop struct{}

func (smsNoop) SendSms(context.Context, string, string, string) error { return nil }
func (smsNoop) CheckBalance(context.Context) (*smsProvider.BalanceResult, error) {
	return &smsProvider.BalanceResult{}, nil
}

type emailNoop struct{}

func (emailNoop) SendEmail(context.Context, string, string, string) error { return nil }

type envelope struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type registerData struct {
	Message string `json:"message"`
	Email   string `json:"email"`
}

type sendOTPData struct {
	OTPTTLSeconds int `json:"otp_ttl_seconds"`
}

type tokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type meData struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

func TestAuthFlow_RegisterSendOTPConfirmOTPMe(t *testing.T) {
	dsn := testDSN(t)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(ctx))

	applySchema(t, pool)

	email := fmt.Sprintf("it_auth_%d@example.com", time.Now().UnixNano())
	phone := fmt.Sprintf("+7999%07d", time.Now().UnixNano()%10000000)
	password := "StrongPass123"
	fullName := "Auth Integration"

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM otp WHERE receiver = $1", email)
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
	})

	router := buildRouter(t, pool)
	ts := httptest.NewServer(router)
	t.Cleanup(ts.Close)

	regPayload := map[string]any{
		"email":     email,
		"phone":     phone,
		"password":  password,
		"full_name": fullName,
		"role":      "volunteer",
	}
	regResp := doJSON(t, http.MethodPost, ts.URL+"/api/v1/register", regPayload, "")
	require.Equal(t, http.StatusOK, regResp.StatusCode)

	var regEnv envelope
	decodeJSON(t, regResp.Body, &regEnv)
	require.Equal(t, "Success", regEnv.Message)
	var reg registerData
	decodeJSON(t, regEnv.Data, &reg)
	require.Equal(t, "verification_required", reg.Message)
	require.Equal(t, email, reg.Email)

	sendResp := doJSON(t, http.MethodPost, ts.URL+"/api/v1/send_otp", map[string]any{
		"receiver": email,
	}, "")
	require.Equal(t, http.StatusOK, sendResp.StatusCode)

	var sendEnv envelope
	decodeJSON(t, sendResp.Body, &sendEnv)
	require.Equal(t, "Success", sendEnv.Message)
	var sendData sendOTPData
	decodeJSON(t, sendEnv.Data, &sendData)
	require.Greater(t, sendData.OTPTTLSeconds, 0)

	var otpCode string
	err = pool.QueryRow(ctx,
		"SELECT otp_code FROM otp WHERE receiver=$1 ORDER BY id DESC LIMIT 1",
		email,
	).Scan(&otpCode)
	require.NoError(t, err)
	require.NotEmpty(t, otpCode)

	confirmResp := doJSON(t, http.MethodPost, ts.URL+"/api/v1/confirm_otp", map[string]any{
		"receiver": email,
		"otp":      otpCode,
	}, "")
	require.Equal(t, http.StatusOK, confirmResp.StatusCode)

	var tokenEnv envelope
	decodeJSON(t, confirmResp.Body, &tokenEnv)
	require.Equal(t, "Success", tokenEnv.Message)
	var tokens tokenData
	decodeJSON(t, tokenEnv.Data, &tokens)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)

	meResp := doJSON(t, http.MethodGet, ts.URL+"/api/v1/me", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, meResp.StatusCode)

	var meEnv envelope
	decodeJSON(t, meResp.Body, &meEnv)
	require.Equal(t, "Success", meEnv.Message)
	var me meData
	decodeJSON(t, meEnv.Data, &me)
	require.Equal(t, email, me.Email)
	require.Equal(t, "volunteer", me.Role)
	require.True(t, me.IsActive)
	require.Greater(t, me.ID, 0)
}

func buildRouter(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()

	logger := zerolog.New(io.Discard)
	repo := repositories.NewRepository(pool, &logger)
	campaignLedgerRepo := repositories.NewCampaignLedgerRepository(pool, slog.Default())

	secret := "integration-test-secret"

	svcCfg := &configs.ServiceConfig{}
	svcCfg.Security.OTPLength = 6
	svcCfg.Security.OTPDuration = 5 * time.Minute
	svcCfg.Security.OTPMaxAttempts = 3
	svcCfg.Security.OTPMaxAttemptsBlockTime = 30 * time.Minute
	svcCfg.Security.SendOTPAttempts = 3
	svcCfg.Security.SendOTPBlockTime = 1 * time.Minute

	svc := services.NewService(
		svcCfg,
		&logger,
		repo,
		campaignLedgerRepo,
		campaignLedgerRepo,
		nil,
		smsNoop{},
		jwtToken.NewJwtTokenIssuer(secret, 15*time.Minute, 720*time.Hour),
		nil,
		emailNoop{},
	)

	cfg := &configs.Config{}
	cfg.App.Env = "test"
	cfg.Security.JWTSecretKey = secret
	cfg.Security.AccessTokenTTL = 15 * time.Minute
	cfg.Security.RefreshTokenTTL = 720 * time.Hour
	cfg.Service = *svcCfg

	h := handlers.NewHandler(svc, allowAllLimiter{}, middlewares.NewMiddleware(secret), &logger, cfg)
	return h.InitRoutes()
}

func doJSON(t *testing.T, method, url string, payload any, accessToken string) *http.Response {
	t.Helper()

	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = resp.Body.Close()
	})
	return resp
}

func decodeJSON(t *testing.T, data any, out any) {
	t.Helper()

	switch v := data.(type) {
	case io.Reader:
		raw, err := io.ReadAll(v)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, out), "body: %s", string(raw))
	case json.RawMessage:
		require.NoError(t, json.Unmarshal(v, out), "raw: %s", string(v))
	default:
		t.Fatalf("unsupported json input type: %T", data)
	}
}

func testDSN(t *testing.T) string {
	t.Helper()

	if dsn := strings.TrimSpace(os.Getenv("TEST_POSTGRES_DSN")); dsn != "" {
		return dsn
	}

	host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	if host == "" {
		host = "localhost"
	}
	if host == "postgres" {
		host = "localhost"
	}

	port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	if port == "" {
		port = "5432"
	}

	user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("POSTGRES_PASSWORD")
	if strings.TrimSpace(password) == "" {
		t.Skip("set TEST_POSTGRES_DSN or POSTGRES_PASSWORD to run integration tests")
	}

	db := strings.TrimSpace(os.Getenv("POSTGRES_DB"))
	if db == "" {
		db = "shb"
	}

	return "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + db + "?sslmode=disable"
}

func applySchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Clean(filepath.Join(cwd, "..", ".."))
	schemaPath := filepath.Join(root, "migration", "shb.sql")
	addOAuthInfoColumnsMigrationPath := filepath.Join(root, "migration", "add_oauth_info_columns.sql")
	executeMigrations(t, pool, schemaPath, addOAuthInfoColumnsMigrationPath)
}

func executeMigrations(t *testing.T, pool *pgxpool.Pool, filePaths ...string) {
	for _, path := range filePaths {
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		_, err = pool.Exec(context.Background(), string(raw))
		require.NoError(t, err)
	}
}
