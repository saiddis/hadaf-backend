// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services_test

import (
	"context"
	"testing"

	"shb/internal/models"
	"shb/pkg/myerrors"

	"github.com/stretchr/testify/require"
)

func TestService_RejectBooking_superAdmin(t *testing.T) {
	ctx := context.Background()
	booking := &models.Booking{ID: 1, NeedID: 20, UserID: 3}
	need := &models.Need{ID: 20, InstitutionID: 5, Name: "n", Unit: "u"}
	svc, d := newTestService(t)
	d.Repo.On("GetBookingByID", ctx, 1).Return(booking, nil)
	d.Repo.On("GetNeedByID", ctx, 20).Return(need, nil)
	d.Repo.On("GetUserByID", ctx, 10).Return(&models.User{ID: 10, Role: models.RoleSuperAdmin}, nil)
	d.Repo.On("UpdateBookingStatus", ctx, 1, models.BookingStatusRejected).Return(nil)
	require.NoError(t, svc.RejectBooking(ctx, 1, 10))
}

func TestService_GetBookingsByInstitution_and_User(t *testing.T) {
	ctx := context.Background()
	list := []*models.Booking{{ID: 1}}

	svc, d := newTestService(t)
	d.Repo.On("GetBookingsByInstitution", ctx, 5).Return(list, nil)
	out, err := svc.GetBookingsByInstitution(ctx, 5)
	require.NoError(t, err)
	require.Equal(t, list, out)

	svc2, d2 := newTestService(t)
	d2.Repo.On("GetBookingsByUser", ctx, 2).Return(list, nil)
	u, err := svc2.GetBookingsByUser(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, list, u)
}

func TestService_CompleteBooking(t *testing.T) {
	ctx := context.Background()
	booking := &models.Booking{ID: 1, NeedID: 20, UserID: 3, Quantity: 2}
	need := &models.Need{ID: 20, InstitutionID: 5, Name: "n", Unit: "u"}
	svc, d := newTestService(t)
	d.Repo.On("GetBookingByID", ctx, 1).Return(booking, nil)
	d.Repo.On("GetNeedByID", ctx, 20).Return(need, nil)
	d.Repo.On("GetUserByID", ctx, 10).Return(&models.User{ID: 10, Role: models.RoleSuperAdmin}, nil)
	// Status update and quantity increment are now a single atomic transaction.
	d.Repo.On("CompleteBookingTx", ctx, 1, 20, 2.0).Return(nil)
	require.NoError(t, svc.CompleteBooking(ctx, 1, 10))
}

func TestService_CancelMyBooking(t *testing.T) {
	ctx := context.Background()
	svc, d := newTestService(t)
	d.Repo.On("GetBookingByID", ctx, 1).Return(&models.Booking{ID: 1, UserID: 2, Status: models.BookingStatusPending}, nil)
	d.Repo.On("UpdateBookingStatus", ctx, 1, "cancelled").Return(nil)
	require.NoError(t, svc.CancelMyBooking(ctx, 1, 2))
}

func TestService_CancelMyBooking_forbidden(t *testing.T) {
	ctx := context.Background()
	svc, d := newTestService(t)
	d.Repo.On("GetBookingByID", ctx, 1).Return(&models.Booking{ID: 1, UserID: 9, Status: models.BookingStatusPending}, nil)
	err := svc.CancelMyBooking(ctx, 1, 2)
	require.Error(t, err)
	var fe myerrors.ForbiddenErr
	require.ErrorAs(t, err, &fe)
}

func TestService_UpdateMyBooking(t *testing.T) {
	ctx := context.Background()
	svc, d := newTestService(t)
	d.Repo.On("GetBookingByID", ctx, 1).Return(&models.Booking{ID: 1, UserID: 2, Status: models.BookingStatusPending}, nil)
	d.Repo.On("UpdateBookingQuantity", ctx, 1, 3.0).Return(nil)
	require.NoError(t, svc.UpdateMyBooking(ctx, 1, 2, 3))
}

func TestService_GetUserByID_RevokeTokens(t *testing.T) {
	ctx := context.Background()
	svc, d := newTestService(t)
	u := &models.User{ID: 1}
	d.Repo.On("GetUserByID", ctx, 1).Return(u, nil)
	out, err := svc.GetUserByID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, u, out)

	d.Repo.On("RevokeAllUserRefreshTokens", ctx, 5).Return(nil)
	require.NoError(t, svc.RevokeAllUserRefreshTokens(ctx, 5))
}
