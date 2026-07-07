// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package repositories

import (
	"shb/internal/models"
)

func (u *dbUser) ToDomain() *models.User {
	if u == nil {
		return nil
	}

	return &models.User{
		ID:                u.ID,
		OAuthProviderName: u.OAuthProviderName,
		OAuthUserID:       u.OAuthUserID,
		AvatarURL:         u.AvatarURL,
		InstitutionID:     u.InstitutionID,
		FullName:          u.FullName,
		Phone:             u.Phone,
		Email:             u.Email,
		Password:          u.Password,
		Role:              u.Role,
		IsActive:          u.IsActive,
		IsApproved:        u.IsApproved,
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
		IsDeleted:         u.IsDeleted,
		DeletedAt:         u.DeletedAt,
	}
}

func (n *dbNeed) ToDomain() *models.Need {
	if n == nil {
		return nil
	}

	return &models.Need{
		ID:            n.ID,
		InstitutionID: n.InstitutionID,
		CategoryID:    n.CategoryID,
		Name:          n.Name,
		Description:   n.Description,
		Unit:          n.Unit,
		RequiredQty:   n.RequiredQty,
		ReceivedQty:   n.ReceivedQty,
		Urgency:       n.Urgency,
		CreatedAt:     n.CreatedAt,
		UpdatedAt:     n.UpdatedAt,
		IsDeleted:     n.IsDeleted,
		DeletedAt:     n.DeletedAt,
	}
}

func (o *dbOtp) ToDomain() *models.OTP {
	if o == nil {
		return nil
	}
	return &models.OTP{
		ID:         o.ID,
		Attempt:    o.Attempt,
		Receiver:   o.Receiver,
		Method:     o.Method,
		OTPCode:    o.OTPCode,
		IsVerified: o.IsVerified,
		SentAt:     o.SentAt,
		ExpiresAt:  o.ExpiresAt,
		UpdatedAt:  o.UpdatedAt,
		IsDeleted:  o.IsDeleted,
		DeletedAt:  o.DeletedAt,
	}
}

func (i *dbInstitution) ToDomain() *models.Institution {
	if i == nil {
		return nil
	}
	return &models.Institution{
		ID:               i.ID,
		Name:             i.Name,
		Type:             i.Type,
		City:             i.City,
		Region:           i.Region,
		Address:          i.Address,
		Phone:            i.Phone,
		Email:            i.Email,
		Description:      i.Description,
		ActivityHours:    i.ActivityHours,
		Latitude:         i.Latitude,
		Longitude:        i.Longitude,
		CreatedAt:        i.CreatedAt,
		UpdatedAt:        i.UpdatedAt,
		IsDeleted:        i.IsDeleted,
		DeletedAt:        i.DeletedAt,
		WardsCount:       i.WardsCount,
		ProhibitedItems:  i.ProhibitedItems,
		RecommendedItems: i.RecommendedItems,
	}
}

func (c *dbCategory) ToDomain() *models.Category {
	if c == nil {
		return nil
	}
	return &models.Category{
		ID:        c.ID,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
		IsDeleted: c.IsDeleted,
		DeletedAt: c.DeletedAt,
	}
}

func (h *dbNeedHistory) ToDomain() *models.NeedsHistory {
	if h == nil {
		return nil
	}
	return &models.NeedsHistory{
		ID:        h.ID,
		NeedID:    h.NeedID,
		Comment:   h.Comment,
		CreatedAt: h.CreatedAt,
		IsDeleted: h.IsDeleted,
		DeletedAt: h.DeletedAt,
	}
}

func (b *dbBooking) ToDomain() *models.Booking {
	if b == nil {
		return nil
	}
	return &models.Booking{
		ID:        b.ID,
		UserID:    b.UserID,
		NeedID:    b.NeedID,
		Quantity:  b.Quantity,
		Note:      b.Note,
		Status:    b.Status,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
		IsDeleted: b.IsDeleted,
		DeletedAt: b.DeletedAt,
	}
}
