// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services

import (
	"context"
	"fmt"
	"shb/internal/models"
)

// UpdateProfile validates the input and partially updates the user's full_name and phone.
func (s *Service) UpdateProfile(ctx context.Context, userID int, req models.UpdateProfileRequest) error {
	if req.FullName == nil && req.Phone == nil {
		return nil
	}

	if err := s.repo.UpdateProfile(ctx, userID, req); err != nil {
		return fmt.Errorf("user service update profile: %w", err)
	}

	return nil
}