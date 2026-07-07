// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package middlewares

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"shb/internal/models"
	"shb/pkg/constants"
	"shb/pkg/myerrors"
	"shb/pkg/notifier"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

// Middleware holds shared middleware state, such as the JWT signing secret.
type Middleware struct {
	jwtSecret        string
	telegramNotifier *notifier.TelegramNotifier
}

// NewMiddleware creates a new Middleware instance with the given JWT secret.
func NewMiddleware(
	jwtSecret string,
	telegramNotifier ...*notifier.TelegramNotifier,
) *Middleware {

	var tn *notifier.TelegramNotifier

	if len(telegramNotifier) > 0 {
		tn = telegramNotifier[0]
	}

	return &Middleware{
		jwtSecret:        jwtSecret,
		telegramNotifier: tn,
	}
}

// AuthMiddleware returns a Gin handler that enforces JWT authentication and,
// optionally, role-based access control. If roles are provided, the caller
// must have at least one of them; otherwise any valid token is accepted.
func (m *Middleware) AuthMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Try Authorization header first, then fall back to httpOnly cookie.
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			headerParts := strings.Split(authHeader, " ")
			if len(headerParts) != 2 || headerParts[0] != "Bearer" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, myerrors.NewUnauthorizedErr("invalid auth header"))
				return
			}
			tokenString = headerParts[1]
		} else if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
			tokenString = cookie
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, myerrors.NewUnauthorizedErr("missing auth credentials"))
			return
		}
		claims := &models.CustomClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, myerrors.NewUnauthorizedErr("invalid token"))
			return
		}

		// Role-based access control.
		if len(roles) > 0 {
			roleAllowed := false
			for _, role := range roles {
				if role == claims.Role {
					roleAllowed = true
					break
				}
			}
			if !roleAllowed {
				c.AbortWithStatusJSON(http.StatusForbidden, myerrors.NewForbiddenErr("access denied"))
				return
			}
		}

		// Employees must be approved by a super-admin before they can access protected routes.
		if claims.Role == models.RoleEmployee && !claims.IsApproved {
			c.AbortWithStatusJSON(http.StatusForbidden, myerrors.NewForbiddenErr("ERR_ACCOUNT_PENDING_APPROVAL"))
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("isApproved", claims.IsApproved)
		c.Next()
	}
}

// OptionalAccessToken is a soft-authentication middleware. It attempts to
// extract and validate the access token but proceeds without error if one is
// absent or invalid, setting userID to 0.
func (m *Middleware) OptionalAccessToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Try Authorization header first, then fall back to httpOnly cookie.
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			headerParts := strings.Split(authHeader, " ")
			if len(headerParts) == 2 && headerParts[0] == "Bearer" {
				tokenString = headerParts[1]
			}
		}
		if tokenString == "" {
			if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
				tokenString = cookie
			}
		}
		if tokenString == "" {
			c.Set("userID", 0)
			c.Next()
			return
		}
		claims := &models.CustomClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.Set("userID", 0)
			c.Next()
			return
		}

		// Unapproved employees are treated as unauthenticated in optional mode.
		if claims.Role == models.RoleEmployee && !claims.IsApproved {
			c.Set("userID", 0)
			c.Next()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("isApproved", claims.IsApproved)
		c.Next()
	}
}

func (m *Middleware) AlertMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		defer func() {

			if err := recover(); err != nil {

				message := fmt.Sprintf(
					` <b>[ PANIC ]</b>

<b>Time:</b> %s
<b>Route:</b> %s %s
<b>Error:</b> %v`,
					time.Now().UTC().Format(time.RFC3339),
					c.Request.Method,
					c.Request.URL.Path,
					err,
				)

				go func() {
					if m.telegramNotifier == nil {
						return
					}

					if sendErr := m.telegramNotifier.SendAlert(message); sendErr != nil {
						log.Printf("telegram alert failed: %v", sendErr)
					}
				}()

				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					gin.H{
						"message": "internal server error",
					},
				)
			}

		}()

		c.Next()

		if c.Writer.Status() >= 500 {

			message := fmt.Sprintf(
				` <b>[ SERVER ERROR ]</b>

<b>Time:</b> %s
<b>Route:</b> %s %s
<b>Status:</b> %d`,
				time.Now().UTC().Format(time.RFC3339),
				c.Request.Method,
				c.Request.URL.Path,
				c.Writer.Status(),
			)

			go func() {
				if m.telegramNotifier != nil {
					if err := m.telegramNotifier.SendAlert(message); err != nil {
						log.Printf("telegram alert failed: %v", err)
					}
				}
			}()
		}
	}
}

// LoggerMiddleware intercepts requests and logs structured data using zerolog,
// pulling the request_id from the request context.
func (m *Middleware) LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		if raw != "" {
			path = path + "?" + raw
		}

		logger := zerolog.Ctx(c.Request.Context())
		status := c.Writer.Status()

		var event *zerolog.Event
		if status >= 500 {
			event = logger.Error()
		} else if status >= 400 {
			event = logger.Warn()
		} else {
			event = logger.Info()
		}

		requestID := c.Writer.Header().Get(constants.RequestIDHeader)
		if requestID == "" {
			requestID = c.GetHeader(constants.RequestIDHeader)
		}

		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", status).
			Str("latency", latency.String()).
			Msg("HTTP Request")
	}
}
