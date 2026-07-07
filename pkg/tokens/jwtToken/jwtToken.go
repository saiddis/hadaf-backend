// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package jwtToken

import (
	"context"
	"fmt"
	"shb/internal/models"
	"shb/pkg/constants"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JwtTokenIssuer struct {
	secretKey  string // Single shared key used by both token issuer and middleware.
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewJwtTokenIssuer creates a JwtTokenIssuer with the given secret and TTL values.
func NewJwtTokenIssuer(secretKey string, accessTTL, refreshTTL time.Duration) *JwtTokenIssuer {
	return &JwtTokenIssuer{
		secretKey:  secretKey,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (j *JwtTokenIssuer) IssueTokens(ctx context.Context, id int, role string, isApproved bool) (string, string, error) {
	now := time.Now().UTC()

	accessClaims := models.CustomClaims{
		UserID:     id,
		Role:       role,
		IsApproved: isApproved,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   constants.AccessSubject,
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        fmt.Sprintf("uid:%d", id),
		},
	}

	refreshClaims := models.CustomClaims{
		UserID:     id,
		Role:       role,
		IsApproved: isApproved,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   constants.RefreshSubject,
			ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        fmt.Sprintf("uid:%d", id),
		},
	}

	// Sign with the same key expected by the middleware.
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString([]byte(j.secretKey))
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
		SignedString([]byte(j.secretKey))
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (j *JwtTokenIssuer) VerifyToken(ctx context.Context, tokenStr string) (*models.CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &models.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	claims, ok := token.Claims.(*models.CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
