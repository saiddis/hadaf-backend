// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"shb/internal/models"
	"shb/pkg/myerrors"
	"shb/pkg/utils"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

// SendOTP generates a one-time password and delivers it to the receiver via
// SMS (for phone numbers) or email. It returns the OTP TTL in seconds.
func (s *Service) SendOTP(ctx context.Context, receiver string) (int, error) {
	otpCode, err := utils.GenerateOTP(s.cfg.Security.OTPLength)
	if err != nil {
		return 0, fmt.Errorf("otp code generation failed: %s", err)
	}

	expiresAt := time.Now().UTC().Add(s.cfg.Security.OTPDuration)
	method := "sms"
	if strings.Contains(receiver, "@") {
		method = "email"
	}

	otp := &models.OTP{
		Receiver:   receiver,
		Method:     &method,
		OTPCode:    otpCode,
		SentAt:     time.Now().UTC(),
		ExpiresAt:  &expiresAt,
		IsVerified: false,
	}

	otpID, err := s.repo.SaveOTP(ctx, otp)
	if err != nil {
		return 0, err
	}

	go func(rcv, code, mthd string, id int) {
		var err error
		if mthd == "email" {
			subject := "Верификационный код Hadaf"
			body := fmt.Sprintf("Никому не передавайте. %s", code)
			err = s.email.SendEmail(context.Background(), rcv, subject, body)
		} else {
			txnID := strconv.Itoa(id)
			err = s.sms.SendSms(context.Background(), rcv, code, txnID)
		}

		if err != nil {
			s.logger.Error().Ctx(ctx).Err(err).
				Str("receiver", rcv).
				Str("method", mthd).
				Msg("failed to send otp")
		} else {
			s.logger.Info().Str("receiver", rcv).Str("method", mthd).Msg("OTP sent successfully")
		}
	}(receiver, otpCode, method, otpID)

	return int(s.cfg.Security.OTPDuration.Seconds()), nil
}

// ConfirmOTP validates the provided OTP code for the given receiver. On
// success it issues a new access/refresh token pair and returns them.
func (s *Service) ConfirmOTP(ctx context.Context, receiver, otp string) (*models.TokenResponse, error) {
	s.logger.Info().
		Str("receiver", receiver).
		Msg("starting OTP confirmation")

	otpDB, err := s.repo.GetOTP(ctx, receiver)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("receiver", receiver).
			Msg("failed to get OTP")

		return nil, myerrors.NewUnauthorizedErr("ERR_OTP_NOT_FOUND_OR_EXPIRED")
	}

	if otp != otpDB.OTPCode {
		s.logger.Warn().
			Str("receiver", receiver).
			Msg("invalid OTP")

		_ = s.repo.IncreaseOTPAttempt(ctx, otpDB.ID, receiver)

		return nil, myerrors.NewUnauthorizedErr("ERR_OTP_INVALID")
	}

	if err = s.repo.MarkOTPAsVerified(ctx, otpDB.ID); err != nil {
		s.logger.Error().
			Err(err).
			Str("receiver", receiver).
			Msg("failed to mark OTP verified")

		return nil, err
	}

	s.logger.Info().
		Str("receiver", receiver).
		Msg("OTP verified")

	var user *models.User

	if strings.Contains(receiver, "@") {
		user, err = s.repo.GetUserByEmail(ctx, receiver)
	} else {
		user, err = s.repo.GetUserByPhone(ctx, receiver)
	}

	// User doesn't exist -> registration flow
	if err != nil && errors.Is(err, myerrors.ErrNotFound) {

		redisKey := fmt.Sprintf("pending_reg:%s", receiver)

		s.logger.Info().
			Str("redis_key", redisKey).
			Msg("user not found, checking pending registration")

		userJSON, cacheErr := s.cache.Get(ctx, redisKey)

		if cacheErr != nil {
			s.logger.Error().
				Err(cacheErr).
				Str("redis_key", redisKey).
				Msg("redis get failed")

			return nil, myerrors.NewBadRequestErr("ERR_USER_NOT_FOUND")
		}

		if userJSON == "" {
			s.logger.Warn().
				Str("redis_key", redisKey).
				Msg("pending registration missing")

			return nil, myerrors.NewBadRequestErr("ERR_USER_NOT_FOUND")
		}

		var pendingUser models.User

		if err := json.Unmarshal([]byte(userJSON), &pendingUser); err != nil {
			s.logger.Error().
				Err(err).
				Msg("failed to decode pending user")

			return nil, fmt.Errorf(
				"failed to deserialize pending user: %w",
				err,
			)
		}

		s.logger.Info().
			Str("receiver", receiver).
			Msg("creating user from pending registration")

		if err := s.repo.CreateUser(ctx, &pendingUser); err != nil {

			s.logger.Error().
				Err(err).
				Str("receiver", receiver).
				Msg("CreateUser failed")

			return nil, fmt.Errorf(
				"failed to create user after otp confirmation: %w",
				err,
			)
		}

		user = &pendingUser

		if err := s.cache.Delete(ctx, redisKey); err != nil {
			s.logger.Warn().
				Err(err).
				Str("redisKey", redisKey).
				Msg("failed to delete pending registration cache")
		}

		s.logger.Info().
			Str("receiver", receiver).
			Msg("user created successfully")

	} else if err != nil {

		s.logger.Error().
			Err(err).
			Str("receiver", receiver).
			Msg("failed to get user")

		return nil, err
	}

	s.logger.Debug().
		Str("role", string(user.Role)).
		Bool("active", user.IsActive).
		Bool("approved", user.IsApproved).
		Msg("user loaded")

	if !user.IsActive {

		s.logger.Info().
			Msg("activating user")

		if err := s.repo.ActivateUser(ctx, user.ID); err != nil {

			s.logger.Error().
				Err(err).
				Msg("failed to activate user")

			return nil, err
		}
	}

	if user.Role == models.RoleEmployee && !user.IsApproved {

		s.logger.Warn().
			Msg("employee waiting for approval")

		return nil, myerrors.NewForbiddenErr(
			"ERR_ACCOUNT_PENDING_APPROVAL",
		)
	}

	access, refresh, err := s.token.IssueTokens(
		ctx,
		user.ID,
		user.Role,
		user.IsApproved,
	)

	if err != nil {

		s.logger.Error().
			Err(err).
			Msg("failed to issue tokens")

		return nil, err
	}

	refreshHash := utils.HashToken(refresh)

	expiresAt := time.Now().
		UTC().
		Add(s.cfg.Security.RefreshTokenTTL)

	if err := s.repo.SaveRefreshToken(
		ctx,
		user.ID,
		refreshHash,
		expiresAt,
	); err != nil {

		s.logger.Error().
			Err(err).
			Msg("failed to save refresh token")

		return nil, fmt.Errorf(
			"failed to save refresh token: %w",
			err,
		)
	}

	s.logger.Info().
		Str("receiver", receiver).
		Msg("OTP confirmation successful")

	return &models.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

// Login authenticates a user by email and password, issuing a new token pair
// on success.
func (s *Service) Login(ctx context.Context, email, password string) (*models.TokenResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, myerrors.ErrNotFound) {
			return nil, myerrors.NewUnauthorizedErr("ERR_INVALID_CREDENTIALS")
		}
		return nil, fmt.Errorf("get user error: %w", err)
	}

	if !user.IsActive {
		return nil, myerrors.NewUnauthorizedErr("ERR_ACCOUNT_NOT_ACTIVATED")
	}

	// Employee approval check.
	if user.Role == models.RoleEmployee && !user.IsApproved {
		return nil, myerrors.NewForbiddenErr("ERR_ACCOUNT_PENDING_APPROVAL")
	}

	if user.Password == nil || *user.Password == "" {
		return nil, myerrors.NewUnauthorizedErr("ERR_INVALID_CREDENTIALS")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return nil, myerrors.NewUnauthorizedErr("ERR_INVALID_CREDENTIALS")
	}

	access, refresh, err := s.token.IssueTokens(ctx, user.ID, user.Role, user.IsApproved)
	if err != nil {
		return nil, fmt.Errorf("issue tokens err: %w", err)
	}

	refreshHash := utils.HashToken(refresh)
	expiresAt := time.Now().UTC().Add(s.cfg.Security.RefreshTokenTTL)
	if err := s.repo.SaveRefreshToken(ctx, user.ID, refreshHash, expiresAt); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &models.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

// LoginOAuth authenticates user by an OAuth provider information, issuing new token pair
// on success.
//
// In case user does not exist, one is created by given data.
func (s *Service) LoginOAuth(ctx context.Context, oauthInfo models.OAuthUserInfo) (*models.TokenResponse, error) {

	if oauthInfo.Email == "" || !oauthInfo.EmailVerified {
		return nil, myerrors.NewUnauthorizedErr("ERR_GOOGLE_EMAIL_NOT_VERIFIED")
	}

	user, err := s.GetUserByEmail(ctx, oauthInfo.Email)
	if err != nil {
		if !errors.Is(err, myerrors.ErrNotFound) {
			return nil, myerrors.NewUnauthorizedErr("ERR_INVALID_CREDENTIALS")
		}
		user = &models.User{
			OAuthUserID:       &oauthInfo.ID,
			OAuthProviderName: &oauthInfo.OAuthProviderName,
			AvatarURL:         oauthInfo.AvatarURL,
			FullName:          &oauthInfo.Username,
			Email:             &oauthInfo.Email,
			Role:              models.RoleVolunteer,
			IsActive:          true,
			IsApproved:        false,
			CreatedAt:         time.Now(),
		}
		if err = s.repo.CreateUser(ctx, user); err != nil {
			return nil, err
		}
	}

	if user.OAuthUserID != nil && *user.OAuthUserID != oauthInfo.ID ||
		user.OAuthProviderName != nil && *user.OAuthProviderName != oauthInfo.OAuthProviderName {
		return nil, myerrors.NewUnauthorizedErr("ERR_OAUTH_ACCOUNT_MISMATCH")
	}

	if user.OAuthUserID == nil || user.OAuthProviderName == nil {
		if user, err = s.repo.UpdateUserOAuthInfoByEmail(ctx, oauthInfo); err != nil {
			return nil, err
		}
	}

	if !user.IsActive {
		return nil, myerrors.NewUnauthorizedErr("ERR_ACCOUNT_NOT_ACTIVATED")
	}

	if user.Role == models.RoleEmployee && !user.IsApproved {
		return nil, myerrors.NewForbiddenErr("ERR_ACCOUNT_PENDING_APPROVAL")
	}

	access, refresh, err := s.token.IssueTokens(ctx, user.ID, user.Role, user.IsApproved)
	if err != nil {
		return nil, fmt.Errorf("issue tokens err: %w", err)
	}

	refreshHash := utils.HashToken(refresh)
	expiresAt := time.Now().UTC().Add(s.cfg.Security.RefreshTokenTTL)
	if err := s.repo.SaveRefreshToken(ctx, user.ID, refreshHash, expiresAt); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &models.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil

}

// RefreshTokens validates the provided refresh token, revokes it, and issues
// a new access/refresh token pair (token rotation).
func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*models.TokenResponse, error) {
	claims, err := s.token.VerifyToken(ctx, refreshToken)
	if err != nil {
		return nil, myerrors.NewUnauthorizedErr("ERR_TOKEN_INVALID")
	}

	refreshHash := utils.HashToken(refreshToken)
	stored, err := s.repo.GetRefreshToken(ctx, refreshHash)
	if err != nil {
		return nil, myerrors.NewUnauthorizedErr("ERR_TOKEN_NOT_FOUND")
	}

	if stored.IsRevoked {
		_ = s.repo.RevokeAllUserRefreshTokens(ctx, stored.UserID)
		return nil, myerrors.NewForbiddenErr("ERR_TOKEN_REVOKED")
	}

	if time.Now().UTC().After(stored.ExpiresAt) {
		return nil, myerrors.NewUnauthorizedErr("ERR_TOKEN_EXPIRED")
	}

	// Rotation: revoke the old token before issuing new ones.
	if err := s.repo.RevokeRefreshToken(ctx, refreshHash); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	access, refresh, err := s.token.IssueTokens(ctx, claims.UserID, claims.Role, claims.IsApproved)
	if err != nil {
		return nil, fmt.Errorf("issue new tokens: %w", err)
	}

	newRefreshHash := utils.HashToken(refresh)
	newExpiresAt := time.Now().UTC().Add(s.cfg.Security.RefreshTokenTTL)
	if err := s.repo.SaveRefreshToken(ctx, claims.UserID, newRefreshHash, newExpiresAt); err != nil {
		return nil, fmt.Errorf("save new refresh token: %w", err)
	}

	return &models.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

// RevokeAllUserRefreshTokens invalidates every active refresh token belonging
// to the given user (used on logout or security events).
func (s *Service) RevokeAllUserRefreshTokens(ctx context.Context, userID int) error {
	return s.repo.RevokeAllUserRefreshTokens(ctx, userID)
}

// Register stores a new user in Redis pending OTP verification. The user
// record is only persisted to the database after ConfirmOTP succeeds.
func (s *Service) Register(ctx context.Context, email, phone, password, fullName, role string, institutionID *int) (*models.TokenResponse, error) {
	log := zerolog.Ctx(ctx).With().
		Str("service", "register").
		Str("email", email).
		Str("phone", phone).
		Logger()

	log.Info().Msg("registration started")
	email = strings.ToLower(strings.TrimSpace(email))
	existing, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, myerrors.NewBadRequestErr("ERR_EMAIL_ALREADY_EXISTS")
	}

	if role == models.RoleEmployee {
		if institutionID == nil {
			return nil, myerrors.NewBadRequestErr("ERR_INSTITUTION_ID_REQUIRED")
		}
		inst, err := s.repo.GetInstitutionByID(ctx, *institutionID)
		if err != nil || inst.IsDeleted {
			return nil, myerrors.NewBadRequestErr("ERR_INSTITUTION_NOT_FOUND")
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	hashedStr := string(hashedPassword)

	newUser := &models.User{
		Email:         &email,
		Phone:         &phone,
		FullName:      &fullName,
		Password:      &hashedStr,
		Role:          role,
		InstitutionID: institutionID,
		IsActive:      true,  // Active immediately; OTP serves as the verification step.
		IsApproved:    false, // Requires super-admin approval.
	}

	// Store in cache until OTP is confirmed; then CreateUser is called.
	userJSON, err := json.Marshal(newUser)
	if err != nil {
		return nil, fmt.Errorf("marshal user: %w", err)
	}

	redisKey := fmt.Sprintf("pending_reg:%s", email)
	if err := s.cache.Set(ctx, redisKey, string(userJSON), 15*time.Minute); err != nil {
		return nil, fmt.Errorf("save user to cache: %w", err)
	}

	if _, err := s.SendOTP(ctx, email); err != nil {
		s.logger.Error().Err(err).Msg("failed to send otp after register")
	}

	return nil, nil
}

// GetUserByID retrieves a user by their primary key.
func (s *Service) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *Service) UserExists(
	ctx context.Context,
	email string,
	phone string,
) (bool, bool, bool, error) {
	var emailExists bool
	var phoneExists bool
	if email != "" {
		_, err := s.repo.GetUserByEmail(ctx, email)
		if err == nil {
			emailExists = true
		} else if !errors.Is(err, myerrors.ErrNotFound) {
			return false, false, false, err
		}
	}
	if phone != "" {
		_, err := s.repo.GetUserByPhone(ctx, phone)

		if err == nil {
			phoneExists = true
		} else if !errors.Is(err, myerrors.ErrNotFound) {
			return false, false, false, err
		}
	}
	return emailExists || phoneExists, emailExists, phoneExists, nil
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.repo.GetUserByEmail(ctx, email)
}

func (s *Service) CreateUser(ctx context.Context, user *models.User) error {
	return s.repo.CreateUser(ctx, user)
}

func (s *Service) UpdateUserOAuthInfoByEmail(ctx context.Context, info models.OAuthUserInfo) (*models.User, error) {
	return s.repo.UpdateUserOAuthInfoByEmail(ctx, info)
}

