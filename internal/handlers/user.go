// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package handlers

import (
	"errors"
	"net/http"
	"shb/internal/models"
	"shb/internal/services"
	"shb/pkg/myerrors"

	"github.com/gin-gonic/gin"
)

// UserHandler handles HTTP requests related to the user profile.
type UserHandler struct {
	service *services.Service
}

// NewUserHandler creates a new user handler instance.
func NewUserHandler(service *services.Service) *UserHandler {
	return &UserHandler{service: service}
}

// UpdateProfile handles PATCH /me requests to update the current user's full name and phone number.
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ERR_UNAUTHORIZED"})
		return
	}
	
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ERR_INTERNAL_SERVER_ERROR"})
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ERR_INVALID_REQUEST_BODY"})
		return
	}

	if err := h.service.UpdateProfile(c.Request.Context(), userID, req); err != nil {
		if errors.Is(err, myerrors.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ERR_USER_NOT_FOUND"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ERR_COULD_NOT_UPDATE_PROFILE"})
		return
	}

	c.Status(http.StatusOK)
}