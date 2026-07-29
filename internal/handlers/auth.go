// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"shb/internal/models"
	"shb/pkg/myerrors"
	"shb/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

func (h *Handler) resetTokenCookies(c *gin.Context) {
	isProduction := h.cfg.App.Env == "production"
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	})
}

// setTokenCookies writes httpOnly cookies for the access and refresh tokens.
func (h *Handler) setTokenCookies(c *gin.Context, tokens *models.TokenResponse) {
	accessMaxAge := int(h.cfg.Security.AccessTokenTTL.Seconds())
	refreshMaxAge := int(h.cfg.Security.RefreshTokenTTL.Seconds())
	isProduction := h.cfg.App.IsProduction()

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    tokens.AccessToken,
		Path:     "/",
		MaxAge:   accessMaxAge,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Path:     "/",
		MaxAge:   refreshMaxAge,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) setOAuthStateCookies(c *gin.Context, state *models.OAuthState) {
	isProduction := h.cfg.App.Env == "production"
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    state.Value,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_return_to",
		Value:    state.ReturnTo,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   h.cfg.App.Env == "production",
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) resetOAuthStateCookies(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.App.Env == "production",
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_return_to",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.App.Env == "production",
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) sendOTP(c *gin.Context) {
	ctx := c.Request.Context()
	logger := h.logger.With().
		Ctx(ctx).
		Str("handler", "sendOTP").
		Logger()

	in := struct {
		Receiver     string `json:"receiver" binding:"required"`
		CaptchaToken string `json:"captcha_token"`
	}{}

	if err := c.ShouldBindJSON(&in); err != nil {
		logger.Warn().Err(err).Msg("invalid request body")
		h.handleError(c, myerrors.NewBadRequestErr("invalid request body"))
		return
	}

	if !strings.Contains(in.Receiver, "@") && !utils.IsValidPhoneNumberByCountry(ctx, in.Receiver) {
		h.handleError(c, myerrors.NewBadRequestErr("invalid email or phone number"))
		return
	}

	key := fmt.Sprintf("user:%s:send_otp", in.Receiver)
	ok, err := h.limiter.Allow(ctx, key, h.cfg.Service.Security.SendOTPAttempts,
		int(h.cfg.Service.Security.SendOTPBlockTime.Minutes()))
	if err != nil {
		logger.Warn().Err(err).Msg("limiter.Allow error")
		h.handleError(c, myerrors.ErrGeneral)
		return
	}
	if !ok {
		logger.Warn().Msg("sendOTP to phone number is out of limit")
		h.handleError(c, myerrors.NewTooManyRequestsErr(
			"phone number is temporarily blocked due to too many requests"))
		return
	}

	ttl, err := h.service.SendOTP(ctx, in.Receiver)
	if err != nil {
		logger.Error().Err(err).Str("phone", in.Receiver).Msg("service.SendOTP error")
		h.handleError(c, err)
		return
	}

	logger.Debug().Str("receiver", in.Receiver).Msg("OTP sent successfully")
	h.success(c, gin.H{
		"otp_ttl_seconds": ttl,
	})
}

func (h *Handler) confirmOTP(c *gin.Context) {
	ctx := c.Request.Context()
	logger := h.logger.With().
		Ctx(ctx).
		Str("handler", "confirmOTP").
		Logger()

	in := struct {
		Receiver string `json:"receiver" binding:"required"`
		OTP      string `json:"otp" binding:"required"`
	}{}

	if err := c.ShouldBindJSON(&in); err != nil {
		logger.Warn().Err(err).Msg("invalid request body")
		h.handleError(c, myerrors.NewBadRequestErr("invalid request body"))
		return
	}

	in.Receiver = strings.ToLower(strings.TrimSpace(in.Receiver))
	in.OTP = strings.TrimSpace(in.OTP)
	if !strings.Contains(in.Receiver, "@") && !utils.IsValidPhoneNumberByCountry(ctx, in.Receiver) {
		logger.Warn().Str("receiver", in.Receiver).Msg("invalid receiver")
		h.handleError(c, myerrors.NewBadRequestErr("invalid receiver format"))
		return
	}

	key := fmt.Sprintf("user:%s:verify_otp", in.Receiver)
	ok, err := h.limiter.Allow(ctx, key, h.cfg.Service.Security.OTPMaxAttempts,
		int(h.cfg.Service.Security.OTPMaxAttemptsBlockTime.Minutes()))
	if err != nil {
		logger.Warn().Err(err).Str("receiver", in.Receiver).Msg("limiter.Allow error")
		h.handleError(c, myerrors.ErrGeneral)
		return
	}
	if !ok {
		logger.Warn().Msg("confirmOTP is out of limit")
		h.handleError(c, myerrors.NewTooManyRequestsErr(
			"receiver is temporarily blocked due to too many requests"))
		return
	}

	response, err := h.service.ConfirmOTP(ctx, in.Receiver, in.OTP)
	if err != nil {
		logger.Error().Err(err).Str("receiver", in.Receiver).Msg("service.ConfirmOTPAndIssueToken error")
		h.handleError(c, err)
		return
	}

	if err = h.limiter.ResetAttempts(ctx, key); err != nil {
		logger.Error().Err(err).Msg("limiter.ResetAttempts error")
	}

	logger.Debug().Str("receiver", in.Receiver).Msg("OTP confirmed successfully")
	h.setTokenCookies(c, response)
	h.success(c, nil) // Tokens are set as httpOnly cookies; not returned in the body.
	logger.Error().
		Err(err).
		Str("receiver", in.Receiver).
		Msg("service.ConfirmOTPAndIssueToken error")

}

func (h *Handler) register(c *gin.Context) {
	ctx := c.Request.Context()

	log := zerolog.Ctx(ctx).
		With().
		Str("handler", "register").
		Logger()

	ctx = log.WithContext(ctx)
	c.Request = c.Request.WithContext(ctx)

	// Apply rate limiting only in non-local environments.
	if !h.cfg.App.IsLocal() {
	isLocal := os.Getenv("APP_ENV") == "development" ||
		os.Getenv("APP_ENV") == "local"

	if !isLocal {
		ipKey := fmt.Sprintf("register_ip:%s", c.ClientIP())
		allowed, err := h.limiter.Allow(ctx, ipKey, 3, 60) // Max 3 registrations per hour per IP.

		if err != nil {
			log.Error().
				Err(err).
				Msg("register rate limiter error")
		}

		if !allowed {
			h.handleError(
				c,
				myerrors.NewTooManyRequestsErr(
					"ERR_RATE_LIMIT_REGISTRATION",
				),
			)
			return
		}
	}

	in := struct {
		Email         string `json:"email" binding:"required,email"`
		Phone         string `json:"phone"`
		Password      string `json:"password" binding:"required"`
		FullName      string `json:"full_name" binding:"required"`
		InstitutionID *int   `json:"institution_id"`
		Role          string `json:"role" binding:"required"`
	}{}

	if err := c.ShouldBindJSON(&in); err != nil {
		log.Warn().
			Err(err).
			Msg("invalid register body")

		h.handleError(
			c,
			myerrors.NewBadRequestErr(
				"invalid input parameters",
			),
		)
		return
	}

	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = strings.TrimSpace(in.Phone)
	in.FullName = strings.TrimSpace(in.FullName)

	if len(in.Password) < 8 {
		h.handleError(
			c,
			myerrors.NewBadRequestErr(
				"password must be at least 8 characters",
			),
		)
		return
	}

	if in.Role != "volunteer" && in.Role != "employee" {
		h.handleError(
			c,
			myerrors.NewBadRequestErr(
				"invalid role: must be volunteer or employee",
			),
		)
		return
	}

	exists, emailExists, phoneExists, err := h.service.UserExists(
		ctx,
		in.Email,
		in.Phone,
	)

	if err != nil {
		log.Error().
			Err(err).
			Msg("checking existing user failed")

		h.handleError(
			c,
			myerrors.ErrGeneral,
		)
		return
	}
	if exists {
		switch {
		case emailExists && phoneExists:
			h.handleError(
				c,
				myerrors.NewConflictErr(
					"email and phone already registered",
				),
			)
		case emailExists:
			h.handleError(
				c,
				myerrors.NewConflictErr(
					"email already registered",
				),
			)
		case phoneExists:
			h.handleError(
				c,
				myerrors.NewConflictErr(
					"phone number already registered",
				),
			)
		}
		return
	}
	_, err = h.service.Register(
		ctx,
		in.Email,
		in.Phone,
		in.Password,
		in.FullName,
		in.Role,
		in.InstitutionID,
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("email", in.Email).
			Msg("registration failed")
		h.handleError(
			c,
			err,
		)
		return
	}
	log.Info().
		Str("email", in.Email).
		Str("role", in.Role).
		Msg("user registration started")
	h.success(
		c,
		gin.H{
			"message": "verification_required",
			"email":   in.Email,
		},
	)
}

func (h *Handler) login(c *gin.Context) {
	ctx := c.Request.Context()
	logger := h.logger.With().Ctx(ctx).Str("handler", "login").Logger()

	in := struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}{}

	if err := c.ShouldBindJSON(&in); err != nil {
		logger.Warn().Err(err).Msg("invalid login input")
		h.handleError(c, myerrors.NewBadRequestErr("invalid request body"))
		return
	}

	// Apply rate limiting only in non-local environments.
	if !h.cfg.App.IsLocal() {
		// Rate limit login — max 5 attempts per 15 minutes per email.
		loginKey := fmt.Sprintf("login:%s", in.Email)
		allowed, err := h.limiter.Allow(ctx, loginKey, 5, 15) // 15 min.
		if err != nil {
			logger.Error().Err(err).Msg("rate limiter error")
		}
		if !allowed {
			h.handleError(c, myerrors.NewTooManyRequestsErr("ERR_RATE_LIMIT_LOGIN"))
			return
		}
	}

	response, err := h.service.Login(ctx, in.Email, in.Password)
	if err != nil {
		logger.Error().Err(err).Str("email", in.Email).Msg("service.Login error")
		h.handleError(c, err)
		return
	}

	logger.Debug().Str("email", in.Email).Msg("login successfully")
	h.setTokenCookies(c, response)
	h.success(c, nil) // Tokens are set as httpOnly cookies; not returned in the body.
}

// refreshTokens handles refresh token rotation.
func (h *Handler) refreshTokens(c *gin.Context) {
	ctx := c.Request.Context()
	logger := h.logger.With().Ctx(ctx).Str("handler", "refreshTokens").Logger()

	// Apply rate limiting only in non-local environments.
	if !h.cfg.App.IsLocal() {
		// Use IP for refresh limiting to prevent endpoint hammering.
		ipKey := fmt.Sprintf("refresh_ip:%s", c.ClientIP())
		allowed, err := h.limiter.Allow(ctx, ipKey, 10, 1) // Max 10 refreshes per minute per IP.
		if err != nil {
			logger.Error().Err(err).Msg("rate limiter error for refresh")
		}
		if !allowed {
			h.handleError(c, myerrors.NewTooManyRequestsErr("ERR_RATE_LIMIT_REFRESH"))
			return
		}
	}

	cookie, err := c.Request.Cookie("refresh_token")
	if err != nil {
		logger.Warn().Msg("refresh token cookie missing")
		h.handleError(c, myerrors.NewUnauthorizedErr("refresh token missing"))
		return
	}

	response, err := h.service.RefreshTokens(ctx, cookie.Value)
	if err != nil {
		logger.Error().Err(err).Msg("service.RefreshTokens error")
		h.handleError(c, err)
		return
	}

	logger.Debug().Msg("tokens refreshed successfully")
	h.setTokenCookies(c, response)
	h.success(c, nil) // Tokens are set as httpOnly cookies; not returned in the body.
}

// logout clears httpOnly auth cookies and revokes all active refresh tokens
// for the user in the database.
func (h *Handler) logout(c *gin.Context) {
	ctx := c.Request.Context()
	log := zerolog.Ctx(ctx).With().Str("handler", "logout").Logger()
	ctx = log.WithContext(ctx)
	c.Request = c.Request.WithContext(ctx)

	userID, exists := c.Get("userID")
	if exists {
		if err := h.service.RevokeAllUserRefreshTokens(ctx, userID.(int)); err != nil {
			log.Error().Err(err).Int("user_id", userID.(int)).Msg("failed to revoke tokens on logout")
		}
	}

	isProduction := h.cfg.App.IsProduction()

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	})
	h.resetTokenCookies(c)

	log.Debug().Msg("user logged out")
	h.success(c, gin.H{"message": "logged out"})
}

// getMe returns the authenticated user's profile.
func (h *Handler) getMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		h.handleError(c, myerrors.NewUnauthorizedErr("user id not found in context"))
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), userID.(int))
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Clear the password hash before sending the response.
	user.Password = nil

	h.success(c, user)
}

func (h *Handler) updateProfile(c *gin.Context) {
	userID, abort := h.mustGetUserID(c)
	if abort {
		return
	}

	in := struct {
		FullName *string `json:"full_name"`
		Phone    *string `json:"phone"`
	}{}

	if err := c.ShouldBindJSON(&in); err != nil {
		h.handleError(c, myerrors.NewBadRequestErr("invalid input parameters"))
		return
	}

	user, err := h.service.UpdateProfile(c.Request.Context(), userID, in.FullName, in.Phone)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Clear the password hash before sending the response.
	user.Password = nil

	h.success(c, user)
// oauth handles the start of server-side OAuth authentication, by redirecting
// user to OAuth provider's consent page.
func (h *Handler) oauth(cfg *oauth2.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := c.Cookie("refresh_token"); err != nil {
			if !errors.Is(err, http.ErrNoCookie) {
				h.handleError(c, err)
				return
			}
			h.resetTokenCookies(c)
		}

		stateBytes := make([]byte, 64)
		if _, err := io.ReadFull(rand.Reader, stateBytes); err != nil {
			return
		}

		state := hex.EncodeToString(stateBytes)

		h.setOAuthStateCookies(c, &models.OAuthState{
			Value:    state,
			ReturnTo: c.Query("return_to"),
		})

		c.Redirect(http.StatusTemporaryRedirect, cfg.AuthCodeURL(state))
	}
}

// oauthCallback handles OAuth provider's callback after a successful user
// consent.
func (h *Handler) oauthCallback(oauthProvider OAuthProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		code, isValid := validateOAuthCallback(c)
		if !isValid {
			h.handleError(c, myerrors.NewForbiddenErr("invalid callback"))
			return
		}

		tok, err := oauthProvider.OAuth2Config().Exchange(c, code)
		if err != nil {
			h.handleError(c, err)
			return
		}

		oauthUserInfo, err := oauthProvider.GetUser(c, tok)
		if err != nil {
			h.handleError(c, err)
			return
		}

		tokens, err := h.service.LoginOAuth(c, oauthUserInfo)
		if err != nil {
			h.handleError(c, err)
			return
		}

		h.setTokenCookies(c, tokens)

		returnTo, err := c.Cookie("oauth_return_to")
		if err != nil {
			h.handleError(c, err)
			return
		}
		h.resetOAuthStateCookies(c)

		redirectURL := h.cfg.App.FrontendURL

		if strings.HasPrefix(returnTo, "/") &&
			!strings.HasPrefix(returnTo, "//") &&
			!strings.Contains(returnTo, "\\") {
			redirectURL += returnTo
		}

		c.Redirect(http.StatusFound, redirectURL)
	}
}

// validateOAuthCallback checks returned oauth state against stored one from
// cookies, and returns authorization code.
func validateOAuthCallback(c *gin.Context) (code string, isValid bool) {
	stateCookie, err := c.Cookie("oauth_state")
	if err != nil {
		return "", false
	}

	if state := c.Query("state"); state == "" || state != stateCookie {
		return "", false
	}

	code = c.Query("code")

	return code, code != ""
}
