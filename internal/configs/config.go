// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package configs

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"shb/pkg/constants"
)

type Config struct {
	App      AppConfig
	Security SecurityConfig
	Database DatabaseConfig
	Logger   LoggerConfig
	SMS      SMSConfig
	SMTP     SMTPConfig
	Server   ServerConfig
	Service  ServiceConfig
	Redis    RedisConfig
	Minio    MinioConfig
	Tracing  TracingConfig
	App         AppConfig
	Security    SecurityConfig
	Database    DatabaseConfig
	Logger      LoggerConfig
	SMS         SMSConfig
	Telegram    TelegramConfig
	SMTP        SMTPConfig
	Server      ServerConfig
	Service     ServiceConfig
	Redis       RedisConfig
	Minio       MinioConfig
	GoogleOAuth OAuthProviderConfig
}

type AppConfig struct {
	FrontendURL string
	Port        string
	Env         string
}

// IsProduction reports whether the app runs in the production environment.
// Used for security-sensitive defaults (e.g. the Secure flag on auth cookies).
func (a AppConfig) IsProduction() bool {
	return a.Env == constants.ProductionAppEnv
}

// IsLocal reports whether the app runs in a local or development environment.
// Used to relax behaviour such as rate limiting during development.
func (a AppConfig) IsLocal() bool {
	return a.Env == constants.LocalAppEnv || a.Env == constants.DevelopmentAppEnv
}

type SecurityConfig struct {
	JWTSecretKey            string
	AccessTokenTTL          time.Duration
	AccessTokenSecret       string
	RefreshTokenTTL         time.Duration
	RefreshTokenSecret      string
	OTPLength               int
	OTPDuration             time.Duration
	OTPMaxAttempts          int
	OTPMaxAttemptsBlockTime time.Duration
	SendOTPAttempts         int
	SendOTPBlockTime        time.Duration
}

type DatabaseConfig struct {
	DSN string
}

type LoggerConfig struct {
	Level         string
	LogPath       string
	IncludeCaller string
}

type SMSConfig struct {
	APIKey     string
	SenderName string
	Login      string
	BaseURL    string
}

type SMTPConfig struct {
	Host      string
	Port      string
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

type ServerConfig struct {
	Name         string
	Port         string
	Host         string
	WriteTimeout string
	ReadTimeout  string
}

type ServiceConfig struct {
	Security SecurityConfig
}

type RedisConfig struct {
	Host      string
	Port      string
	DefaultDB int
	Timeout   string
}

type MinioConfig struct {
	Bucket    string
	Endpoint  string
	AccessKey string
	SecretKey string
}

// TracingConfig holds the OpenTelemetry tracing settings (Phase 3). Tracing is
// opt-in: when Enabled is false the SDK installs a no-op provider and the app
// emits no spans, so the default deployment is unchanged.
type TracingConfig struct {
	Enabled        bool
	Endpoint       string  // OTLP/gRPC Collector endpoint, e.g. "otel-collector:4317"
	Insecure       bool    // disable TLS (true for in-network transport; prefer mTLS in prod)
	ServiceName    string  // resource attribute service.name
	ServiceVersion string  // resource attribute service.version
	SampleRatio    float64 // head sampling probability in [0,1]
type TelegramConfig struct {
	BaseURL string
	Token   string
	ChatID  string
}

type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// Helper to read ENV with a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// getEnvBool reads a boolean ENV var, falling back when unset or unparseable.
func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

// getEnvFloat reads a float ENV var, falling back when unset or unparseable.
func getEnvFloat(key string, fallback float64) float64 {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

// requireEnv panics if env variable is missing or empty (for secrets)
func requireEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		panic(fmt.Sprintf("FATAL: required env var %s is not set", key))
	}
	return v
}

// InitConfigs loads the configuration
func InitConfigs() (*Config, error) {
	// Read PostgreSQL variables
	pgUser := getEnv("POSTGRES_USER", "postgres")
	pgPass := requireEnv("POSTGRES_PASSWORD")
	pgHost := getEnv("POSTGRES_HOST", "localhost") // "postgres" in Docker
	pgPort := getEnv("POSTGRES_PORT", "5432")
	pgDB := getEnv("POSTGRES_DB", "shb")

	// Build DSN string
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		pgUser, pgPass, pgHost, pgPort, pgDB)

	// Read Redis configuration
	// Note: If redisClient reads REDIS_HOST via os.Getenv directly, it will work.
	// If it takes the config from here, we aren't passing it explicitly into the struct yet,
	// but the presence of variables in ENV (via docker-compose) will suffice.

	// Default to production so an unset APP_ENV is secure-by-default (Secure
	// cookies on, Swagger off). The legacy "prod" value is normalised to the
	// canonical "production" so existing deployments keep working correctly.
	appEnv := getEnv("APP_ENV", constants.ProductionAppEnv)
	if appEnv == "prod" {
		appEnv = constants.ProductionAppEnv
	}

	security := SecurityConfig{
		JWTSecretKey:            requireEnv("JWT_SECRET_KEY"),
		AccessTokenTTL:          15 * time.Minute,
		AccessTokenSecret:       requireEnv("ACCESS_TOKEN_SECRET"),
		RefreshTokenTTL:         720 * time.Hour,
		RefreshTokenSecret:      requireEnv("REFRESH_TOKEN_SECRET"),
		OTPLength:               6,
		OTPDuration:             5 * time.Minute,
		OTPMaxAttempts:          3,
		OTPMaxAttemptsBlockTime: 30 * time.Minute,
		SendOTPAttempts:         3,
		SendOTPBlockTime:        1 * time.Minute,
	}

	return &Config{
		App: AppConfig{
			Port: getEnv("APP_PORT", ":8000"),
			Env:  appEnv,
			Port:        getEnv("APP_PORT", ":8000"),
			Env:         getEnv("APP_ENV", "prod"),
			FrontendURL: getEnv("APP_FRONTEND_URL", "http://localhost:3000"),
		},
		Security: security,
		Database: DatabaseConfig{
			DSN: dsn,
		},
		Logger: LoggerConfig{
			Level:         getEnv("LOG_LEVEL", "debug"),
			LogPath:       getEnv("LOG_PATH", ""),
			IncludeCaller: getEnv("INCLUDE_CALLER", "false"),
		},
		SMS: SMSConfig{
			APIKey:     getEnv("SMS_API_KEY", "mock"),
			SenderName: getEnv("SMS_SENDER_NAME", "Payvand"),
			Login:      getEnv("SMS_LOGIN", ""),
			BaseURL:    getEnv("SMS_BASE_URL", "https://api.osonsms.com"),
		},
		SMTP: SMTPConfig{
			Host:      getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port:      getEnv("SMTP_PORT", "587"),
			Username:  getEnv("SMTP_USERNAME", ""),
			Password:  getEnv("SMTP_PASSWORD", ""),
			FromEmail: getEnv("SMTP_FROM_EMAIL", "noreply@socialhousing.tj"),
			FromName:  getEnv("SMTP_FROM_NAME", "Social Housing Platform"),
		},
		Server: ServerConfig{
			Name:         "SocialHousingBackend",
			Port:         getEnv("APP_PORT", ":8000"),
			Host:         getEnv("APP_HOST", "localhost"),
			WriteTimeout: getEnv("APP_WRITE_TIMEOUT", "10s"),
			ReadTimeout:  getEnv("APP_READ_TIMEOUT", "10s"),
		},
		Service: ServiceConfig{
			Security: security,
		},
		Redis: RedisConfig{
			Host:      getEnv("REDIS_HOST", "localhost"),
			Port:      getEnv("REDIS_PORT", "6379"),
			DefaultDB: 0,
			Timeout:   getEnv("REDIS_TIMEOUT", "5s"),
		},
		Minio: MinioConfig{
			Bucket:    getEnv("MINIO_BUCKET", "minio"),
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "minio"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "minio"),
		},
		Tracing: TracingConfig{
			Enabled:        getEnvBool("OTEL_TRACES_ENABLED", false),
			Endpoint:       getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317"),
			Insecure:       getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", true),
			ServiceName:    getEnv("OTEL_SERVICE_NAME", "shb"),
			ServiceVersion: getEnv("OTEL_SERVICE_VERSION", "dev"),
			SampleRatio:    getEnvFloat("OTEL_TRACES_SAMPLER_ARG", 1.0),
		Telegram: TelegramConfig{
			Token:   getEnv("TELEGRAM_ALERT_TOKEN", ""),
			ChatID:  getEnv("TELEGRAM_ALERT_CHAT_ID", ""),
			BaseURL: getEnv("TELEGRAM_BASE_URL", "https://api.telegram.org"),
		},
		GoogleOAuth: OAuthProviderConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
		},
	}, nil
}
