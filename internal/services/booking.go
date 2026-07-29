// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services

import (
	"context"
	"errors"
	"fmt"

	"shb/internal/models"
	"shb/pkg/myerrors"

	"github.com/rs/zerolog"
)

// CreateBooking registers a volunteer's intent to fulfill a specific need.
// It validates the need, the user state, and prevents duplicate active bookings.
// On success it sends an email notification to the institution asynchronously.
func (s *Service) CreateBooking(ctx context.Context, userID, needID int, quantity float64, note string) (int, error) {
	log := zerolog.Ctx(ctx).With().Str("service", "CreateBooking").Int("user_id", userID).Int("need_id", needID).Logger()

	need, err := s.repo.GetNeedByID(ctx, needID)
	if err != nil {
		if errors.Is(err, myerrors.ErrNotFound) {
			return 0, myerrors.NewBadRequestErr("need not found")
		}
		return 0, fmt.Errorf("get need: %w", err)
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, myerrors.ErrNotFound) {
			return 0, myerrors.NewBadRequestErr("user not found")
		}
		return 0, fmt.Errorf("get user: %w", err)
	}
	if !user.IsActive {
		return 0, myerrors.NewBadRequestErr("user is not active")
	}

	if quantity <= 0 {
		return 0, myerrors.NewBadRequestErr("quantity must be greater than 0")
	}

	existingBooking, err := s.repo.GetActiveBookingByUserAndNeed(ctx, userID, needID)
	if err != nil {
		return 0, fmt.Errorf("check existing booking: %w", err)
	}
	if existingBooking != nil {
		return 0, myerrors.NewConflictErr("ERR_BOOKING_ALREADY_EXISTS")
	}

	booking := &models.Booking{
		UserID:   userID,
		NeedID:   needID,
		Quantity: quantity,
		Note:     note,
		Status:   models.BookingStatusPending,
	}

	bookingID, err := s.repo.CreateBooking(ctx, booking)
	if err != nil {
		return 0, fmt.Errorf("create booking: %w", err)
	}

	log.Info().Int("booking_id", bookingID).Float64("quantity", quantity).Msg("booking created")

	institution, err := s.repo.GetInstitutionByID(ctx, need.InstitutionID)
	if err != nil {
		log.Error().Err(err).Int("institution_id", need.InstitutionID).Msg("failed to get institution for email notification")
		return bookingID, nil
	}

	if institution.Email != nil && *institution.Email != "" {
		userPhone := ""
		if user.Phone != nil {
			userPhone = *user.Phone
		}
		userFullName := ""
		if user.FullName != nil {
			userFullName = *user.FullName
		}

		subject := "New volunteer is ready to help"
		body := fmt.Sprintf(`Institution: %s
Need: %s
Volunteer: %s
Phone: %s
Quantity: %.2f %s
Message: %s

Please contact the volunteer to coordinate.`,
			institution.Name, need.Name, userFullName, userPhone, quantity, need.Unit, note,
		)

		if err := s.email.SendEmail(ctx, *institution.Email, subject, body); err != nil {
			log.Error().Err(err).Str("email", *institution.Email).Msg("failed to send booking notification email")
		}
	}

	return bookingID, nil
}

// ApproveBooking marks a booking as approved. Only employees of the owning
// institution or super-admins may perform this action.
func (s *Service) ApproveBooking(ctx context.Context, bookingID, institutionUserID int) error {
	log := zerolog.Ctx(ctx).With().Str("service", "ApproveBooking").Int("booking_id", bookingID).Int("actor_id", institutionUserID).Logger()

	booking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("get booking: %w", err)
	}

	need, err := s.repo.GetNeedByID(ctx, booking.NeedID)
	if err != nil {
		return fmt.Errorf("get need: %w", err)
	}

	requester, err := s.repo.GetUserByID(ctx, institutionUserID)
	if err != nil {
		return fmt.Errorf("get requester user: %w", err)
	}

	if requester.Role != models.RoleSuperAdmin && requester.Role != models.RoleEmployee {
		return myerrors.NewForbiddenErr("only employees and super admins can approve bookings")
	}

	if requester.Role == models.RoleEmployee {
		if requester.InstitutionID == nil || *requester.InstitutionID != need.InstitutionID {
			return myerrors.NewForbiddenErr("you can only approve bookings for your own institution")
		}
	}

	if err := s.repo.UpdateBookingStatus(ctx, bookingID, models.BookingStatusApproved); err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}

	log.Info().Msg("booking approved")
	return nil
}

// RejectBooking marks a booking as rejected. Only employees of the owning
// institution or super-admins may perform this action.
func (s *Service) RejectBooking(ctx context.Context, bookingID, institutionUserID int) error {
	log := zerolog.Ctx(ctx).With().Str("service", "RejectBooking").Int("booking_id", bookingID).Int("actor_id", institutionUserID).Logger()

	booking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("get booking: %w", err)
	}

	need, err := s.repo.GetNeedByID(ctx, booking.NeedID)
	if err != nil {
		return fmt.Errorf("get need: %w", err)
	}

	requester, err := s.repo.GetUserByID(ctx, institutionUserID)
	if err != nil {
		return fmt.Errorf("get requester user: %w", err)
	}

	if requester.Role != models.RoleSuperAdmin && requester.Role != models.RoleEmployee {
		return myerrors.NewForbiddenErr("only employees and super admins can reject bookings")
	}

	if requester.Role == models.RoleEmployee {
		if requester.InstitutionID == nil || *requester.InstitutionID != need.InstitutionID {
			return myerrors.NewForbiddenErr("you can only reject bookings for your own institution")
		}
	}

	if err := s.repo.UpdateBookingStatus(ctx, bookingID, models.BookingStatusRejected); err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}

	log.Info().Msg("booking rejected")
	return nil
}

// CompleteBooking marks a booking as completed and increments the need's
// received quantity. Only employees of the owning institution or super-admins
// may perform this action.
func (s *Service) CompleteBooking(ctx context.Context, bookingID, institutionUserID int) error {
	log := zerolog.Ctx(ctx).With().Str("service", "CompleteBooking").Int("booking_id", bookingID).Int("actor_id", institutionUserID).Logger()

	booking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("get booking: %w", err)
	}

	need, err := s.repo.GetNeedByID(ctx, booking.NeedID)
	if err != nil {
		return fmt.Errorf("get need: %w", err)
	}

	requester, err := s.repo.GetUserByID(ctx, institutionUserID)
	if err != nil {
		return fmt.Errorf("get requester user: %w", err)
	}

	if requester.Role != models.RoleSuperAdmin && requester.Role != models.RoleEmployee {
		return myerrors.NewForbiddenErr("only employees and super admins can complete bookings")
	}

	if requester.Role == models.RoleEmployee {
		if requester.InstitutionID == nil || *requester.InstitutionID != need.InstitutionID {
			return myerrors.NewForbiddenErr("you can only complete bookings for your own institution")
		}
	}

	// Status update and quantity increment must be atomic: a completed booking
	// whose quantity was not applied (or vice versa) leaves the need's totals
	// permanently inconsistent.
	if err := s.repo.CompleteBookingTx(ctx, bookingID, booking.NeedID, booking.Quantity); err != nil {
		return fmt.Errorf("complete booking: %w", err)
	}

	log.Info().Float64("quantity", booking.Quantity).Msg("booking completed")
	return nil
}

// GetBookingsByInstitution returns all bookings associated with the given
// institution's needs.
func (s *Service) GetBookingsByInstitution(ctx context.Context, institutionID int) ([]*models.Booking, error) {
	bookings, err := s.repo.GetBookingsByInstitution(ctx, institutionID)
	if err != nil {
		return nil, fmt.Errorf("get bookings by institution: %w", err)
	}
	return bookings, nil
}

// GetBookingsByUser returns all bookings created by the given user.
func (s *Service) GetBookingsByUser(ctx context.Context, userID int) ([]*models.Booking, error) {
	bookings, err := s.repo.GetBookingsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get bookings by user: %w", err)
	}
	return bookings, nil
}

// CancelMyBooking allows a volunteer to cancel their own pending booking.
func (s *Service) CancelMyBooking(ctx context.Context, bookingID int, userID int) error {
	log := zerolog.Ctx(ctx).With().Str("service", "CancelMyBooking").Int("booking_id", bookingID).Int("user_id", userID).Logger()

	booking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("get booking: %w", err)
	}
	if booking.UserID != userID {
		return myerrors.NewForbiddenErr("you can only cancel your own bookings")
	}
	if booking.Status != models.BookingStatusPending {
		return myerrors.NewBadRequestErr("only pending bookings can be cancelled")
	}

	if err := s.repo.UpdateBookingStatus(ctx, bookingID, "cancelled"); err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}

	log.Info().Msg("booking cancelled")
	return nil
}

// UpdateMyBooking allows a volunteer to change the quantity on their own
// pending booking.
func (s *Service) UpdateMyBooking(ctx context.Context, bookingID int, userID int, qty float64) error {
	log := zerolog.Ctx(ctx).With().Str("service", "UpdateMyBooking").Int("booking_id", bookingID).Int("user_id", userID).Logger()

	booking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("get booking: %w", err)
	}
	if booking.UserID != userID {
		return myerrors.NewForbiddenErr("you can only modify your own bookings")
	}
	if booking.Status != models.BookingStatusPending {
		return myerrors.NewBadRequestErr("only pending bookings can be modified")
	}
	if qty <= 0 {
		return myerrors.NewBadRequestErr("quantity must be greater than 0")
	}

	if err := s.repo.UpdateBookingQuantity(ctx, bookingID, qty); err != nil {
		return fmt.Errorf("update booking quantity: %w", err)
	}

	log.Info().Float64("quantity", qty).Msg("booking quantity updated")
	return nil
}
