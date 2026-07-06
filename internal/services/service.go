// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services

import (
	"context"
	"time"

	"shb/internal/configs"
	"shb/internal/models"
	"shb/internal/repositories/filters"
	"shb/pkg/db/cache"
	"shb/pkg/external/email"
	"shb/pkg/external/fs"
	"shb/pkg/external/sms"
	"shb/pkg/tokens"

	"github.com/rs/zerolog"
)

// IRepository defines the data access layer contract used by the service layer.
type IRepository interface {
	// GetUserByPhone returns the user that matches the given phone number.
	GetUserByPhone(ctx context.Context, phone string) (*models.User, error)
	// GetUserByEmail returns the user that matches the given email address.
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	// GetUserByID returns the user identified by the given primary key.
	GetUserByID(ctx context.Context, id int) (*models.User, error)
	// CreateUser persists a new user record to the database.
	CreateUser(ctx context.Context, user *models.User) error
	// ActivateUser marks the user account as active.
	ActivateUser(ctx context.Context, id int) error
	// GetUserByOAuthInfo return the user that matches the given OAuth user id and provider name.
	GetUserByOAuthInfo(ctx context.Context, oauthUserID, oauthProviderName string) (*models.User, error)
	// UpdateUserOAuthInfoByEmail updates user's oauth provider name and oauth user id by email from info
	UpdateUserOAuthInfoByEmail(ctx context.Context, info models.OAuthUserInfo) (*models.User, error)
	// UpdateProfile updates the full_name and phone fields of a user if provided.
	UpdateProfile(ctx context.Context, id int, req models.UpdateProfileRequest) error
	// SaveOTP persists a new OTP record to the database.
	SaveOTP(ctx context.Context, o *models.OTP) (int, error)
	// GetOTP retrieves the latest active, unverified OTP for the given receiver.
	GetOTP(ctx context.Context, phone string) (*models.OTP, error)
	// MarkOTPAsVerified marks an OTP record as verified.
	MarkOTPAsVerified(ctx context.Context, otpID int) error
	// IncreaseOTPAttempt increments the failed-attempt counter on an OTP record.
	IncreaseOTPAttempt(ctx context.Context, otpID int, phone string) error

	// --- Institution Methods ---
	GetAllInstitutions(ctx context.Context, q models.InstitutionListQuery) (*models.InstitutionPage, error)
	CreateInstitution(ctx context.Context, i *models.Institution) (int, error)
	GetInstitutionByID(ctx context.Context, id int) (*models.Institution, error)

	// --- Needs Methods ---
	CreateNeed(ctx context.Context, need *models.Need) (int, error)
	GetNeedByID(ctx context.Context, id int) (*models.Need, error)
	UpdateNeed(ctx context.Context, n *models.Need) error
	DeleteNeed(ctx context.Context, id int) error
	GetNeedsByInstitution(ctx context.Context, filter filters.NeedsFilter, institutionID int) ([]*models.Need, error)

	// --- Booking Methods ---
	CreateBooking(ctx context.Context, booking *models.Booking) (int, error)
	GetBookingByID(ctx context.Context, id int) (*models.Booking, error)
	GetBookingsByNeed(ctx context.Context, needID int) ([]*models.Booking, error)
	GetBookingsByUser(ctx context.Context, userID int) ([]*models.Booking, error)
	GetActiveBookingByUserAndNeed(ctx context.Context, userID, needID int) (*models.Booking, error)
	UpdateBookingStatus(ctx context.Context, bookingID int, status string) error
	UpdateBookingQuantity(ctx context.Context, bookingID int, qty float64) error
	IncrementReceivedQty(ctx context.Context, needID int, qty float64) error
	GetBookingsByInstitution(ctx context.Context, institutionID int) ([]*models.Booking, error)

	// --- Event Methods ---
	CreateEvent(ctx context.Context, e *models.Event) (int, error)
	GetEventByID(ctx context.Context, id int) (*models.Event, error)
	GetEventDetail(ctx context.Context, q models.EventDetailQuery) (*models.EventResponse, error)
	GetAllEvents(ctx context.Context, q models.EventListQuery) (*models.EventPage, error)
	JoinEvent(ctx context.Context, eventID, userID int) error
	LeaveEvent(ctx context.Context, eventID, userID int) error
	GetInstitutionEvents(ctx context.Context, institutionID int) ([]*models.EventResponse, error)
	UpdateEventStatus(ctx context.Context, eventID int, status string) error

	// --- Vacancies ---
	GetAllVacancies(ctx context.Context) ([]*models.Vacancy, error)
	GetVacancyByID(ctx context.Context, id int) (*models.Vacancy, error)

	// --- Team Members ---
	GetAllTeamMembers(ctx context.Context) ([]*models.TeamMember, error)
	GetTeamMemberByID(ctx context.Context, id int) (*models.TeamMember, error)

	// --- Stats ---
	GetPublicStats(ctx context.Context) (map[string]int, error)

	CreateNeedHistory(ctx context.Context, history *models.NeedsHistory) error

	// --- Token Methods ---
	SaveRefreshToken(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID int) error
}

// Service is the application service layer that coordinates business logic
// across the repository, cache, external adapters, and token provider.
type Service struct {
	cfg    *configs.ServiceConfig
	logger *zerolog.Logger
	repo   IRepository
	cache  cache.ICache
	sms    sms.ISmsAdapter
	token  tokens.ITokenIssuer
	fs     fs.Storage
	email  email.IEmailAdapter
}

// NewService constructs a Service with all required dependencies injected.
func NewService(cfg *configs.ServiceConfig, log *zerolog.Logger, repo IRepository, cache cache.ICache,
	sms sms.ISmsAdapter, token tokens.ITokenIssuer, fs fs.Storage, email email.IEmailAdapter) *Service {
	return &Service{
		cfg:    cfg,
		logger: log,
		repo:   repo,
		cache:  cache,
		sms:    sms,
		email:  email,
		token:  token,
		fs:     fs,
	}
}
