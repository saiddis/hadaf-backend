// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package application

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"shb/internal/configs"
	"shb/internal/handlers"
	"shb/internal/oauth"
	"shb/internal/repositories"
	"shb/internal/server"
	"shb/internal/services"
	"shb/pkg/db/cache/redisClient"
	"shb/pkg/db/pgx"
	"shb/pkg/external/email/smtpEmail"
	"shb/pkg/external/fs/minioFs"
	"shb/pkg/external/sms/smsProvider"
	"shb/pkg/logger"
	"shb/pkg/metrics"
	"shb/pkg/middlewares"
	"shb/pkg/notifier"
	"shb/pkg/rateLimiter/customLimiter"
	"shb/pkg/tokens/jwtToken"
	"shb/pkg/tracing"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"
)

type App struct {
	config         *configs.Config
	logger         *logger.Logger
	server         *server.Server
	tracerShutdown func(context.Context) error
}

func NewApplication() *App {
	// 1. Load Internal Config
	cfg, err := configs.InitConfigs()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}
	// 2. Map Internal Config to Pkg Config for Logger.
	loggerCfg := logger.Config{
		Level:         cfg.Logger.Level,
		Env:           cfg.App.Env,
		LogPath:       cfg.Logger.LogPath,
		IncludeCaller: cfg.Logger.IncludeCaller == "true",
	}
	log, err := logger.NewLogger(loggerCfg)
	if err != nil {
		fallbackCfg := loggerCfg
		fallbackCfg.LogPath = ""
		log, err = logger.NewLogger(fallbackCfg)
		if err != nil {
			panic("failed to initialize logger: " + err.Error())
		}
		log.Warn().
			Str("log_path", loggerCfg.LogPath).
			Msg("could not open log file, falling back to stdout logging")
	}

	// Observability: initialise the OpenTelemetry tracer provider before any
	// instrumented dependency is constructed. Disabled by default — when off it
	// installs a no-op provider, so nothing downstream changes behaviour.
	tracerShutdown, err := tracing.Init(context.Background(), tracing.Config{
		Enabled:        cfg.Tracing.Enabled,
		Endpoint:       cfg.Tracing.Endpoint,
		Insecure:       cfg.Tracing.Insecure,
		ServiceName:    cfg.Tracing.ServiceName,
		ServiceVersion: cfg.Tracing.ServiceVersion,
		Environment:    cfg.App.Env,
		SampleRatio:    cfg.Tracing.SampleRatio,
	})
	if err != nil {
		// A tracing misconfiguration must not take the service down — log and
		// continue with the no-op provider that Init returns on error.
		log.Warn().Err(err).Msg("tracing initialisation failed; continuing without traces")
	}
	if cfg.Tracing.Enabled {
		log.Info().
			Str("endpoint", cfg.Tracing.Endpoint).
			Float64("sample_ratio", cfg.Tracing.SampleRatio).
			Msg("OpenTelemetry tracing enabled")
	}
	log.Info().
		Str("smtp_user", cfg.SMTP.Username).
		Bool("smtp_password_set", cfg.SMTP.Password != "").
		Msg("smtp config loaded")

	m, err := migrate.New("file://migration", cfg.Database.DSN)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize migrations")
	} else {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal().Err(err).Msg("failed to apply migrations")
		} else {
			log.Info().Msg("database migrations applied successfully")
		}
	}

	postgresConn, err := pgx.NewPgxPool()
	if err != nil {
		panic("failed to connect to postgres: " + err.Error())
	}

	redis, err := redisClient.NewRedisClient()
	if err != nil {
		panic("failed to connect to redis: " + err.Error())
	}

	fileStorage, err := minioFs.NewMinIOStorage(minioFs.MinIOConfig{
		Bucket:    cfg.Minio.Bucket,
		Endpoint:  cfg.Minio.Endpoint,
		AccessKey: cfg.Minio.AccessKey,
		SecretKey: cfg.Minio.SecretKey,
		Logger:    &log.Logger,
	})
	if err != nil {
		panic("failed to initialize minio storage: " + err.Error())
	}

	limiter := customLimiter.NewRateLimiter(redis)

	// 3. SMS (Map config)
	sms := smsProvider.NewSMSProvider(smsProvider.SMSConfig{
		APIKey:     cfg.SMS.APIKey,
		SenderName: cfg.SMS.SenderName,
		Login:      cfg.SMS.Login,
		BaseURL:    cfg.SMS.BaseURL,
	}, &log.Logger)

	// 4. Initialize SMTP Email Adapter
	emailAdapter := smtpEmail.NewSMTPEmail(&cfg.SMTP)

	log.Info().
		Bool(
			"telegram_alerts_enabled",
			cfg.Telegram.Token != "" && cfg.Telegram.ChatID != "",
		).
		Msg("telegram notifier initialized")

	token := jwtToken.NewJwtTokenIssuer(
		cfg.Security.JWTSecretKey,
		cfg.Security.AccessTokenTTL,
		cfg.Security.RefreshTokenTTL,
	)

	// 4. Middleware (Pass secret)
	telegramNotifier := notifier.NewTelegramNotifier(cfg.Telegram)

	middleware := middlewares.NewMiddleware(
		cfg.Security.JWTSecretKey,
		telegramNotifier,
	)

	repository := repositories.NewRepository(postgresConn, &log.Logger)

	// Observability: create the metrics registry first, then wrap the external
	// adapters so every outbound call (SMS, email, object storage) is timed.
	appMetrics := metrics.New()
	appMetrics.RegisterDBPool(postgresConn)

	instrumentedSMS := appMetrics.InstrumentSMS(sms)
	instrumentedEmail := appMetrics.InstrumentEmail(emailAdapter)
	instrumentedStorage := appMetrics.InstrumentStorage(fileStorage)

	// Service uses Internal Config (ServiceConfig)
	// We pass emailAdapter here as it was required by Service constructor
	service := services.NewService(&cfg.Service, &log.Logger, repository, redis,
		instrumentedSMS, token, instrumentedStorage, instrumentedEmail)

	handler := handlers.NewHandler(service, limiter, middleware, appMetrics, &log.Logger, cfg)
	googleOAuthProvider := oauth.NewGoogleProvider(&cfg.GoogleOAuth)

	handler := handlers.NewHandler(service, limiter, middleware, &log.Logger, cfg, googleOAuthProvider)

	// 5. Server (Map config)
	readTimeout, _ := time.ParseDuration(cfg.Server.ReadTimeout)
	writeTimeout, _ := time.ParseDuration(cfg.Server.WriteTimeout)

	srv := server.NewServer(server.Config{
		Port:         cfg.Server.Port,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}, handler.InitRoutes()) // Note: NewServer in pkg/server now accepts an http.Handler

	return &App{
		config:         cfg,
		logger:         log,
		server:         srv,
		tracerShutdown: tracerShutdown,
	}
}

func (a *App) Start() {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		a.logger.Info().Msgf("%s is started on port %s", a.config.Server.Name, a.config.Server.Port)
		if err := a.server.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logger.Fatal().Msgf("Could not listen: %v", err)
		}
	}()

	<-stopChan
	a.logger.Info().Msg("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Fatal().Msgf("Server forced to shutdown: %v", err)
	}

	// Flush any buffered spans to the Collector before exit. Use a fresh
	// timeout: the server-shutdown context above may already be (nearly)
	// exhausted, which would abort the span flush.
	if a.tracerShutdown != nil {
		flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer flushCancel()
		if err := a.tracerShutdown(flushCtx); err != nil {
			a.logger.Warn().Err(err).Msg("tracer shutdown error")
		}
	}

	a.logger.Info().Msg("Server exited properly")
}
