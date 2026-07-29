// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"shb/internal/models"
	"shb/pkg/myerrors"

	"github.com/stretchr/testify/require"
)

func strptr(s string) *string { return &s }

func TestService_UpdateProfile(t *testing.T) {
	ctx := context.Background()
	const userID = 7

	tests := []struct {
		name      string
		fullName  *string
		phone     *string
		setup     func(d testDeps)
		wantErr   bool
		assertErr func(t *testing.T, err error)
		assert    func(t *testing.T, u *models.User)
	}{
		{
			name:    "no fields provided",
			wantErr: true,
			setup:   func(d testDeps) {},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				var br myerrors.BadRequestErr
				require.ErrorAs(t, err, &br)
			},
		},
		{
			name:     "blank full name",
			fullName: strptr("   "),
			wantErr:  true,
			setup:    func(d testDeps) {},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				var br myerrors.BadRequestErr
				require.ErrorAs(t, err, &br)
			},
		},
		{
			name:     "full name too long",
			fullName: strptr(strings.Repeat("a", 151)),
			wantErr:  true,
			setup:    func(d testDeps) {},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				var br myerrors.BadRequestErr
				require.ErrorAs(t, err, &br)
			},
		},
		{
			name:    "blank phone",
			phone:   strptr("  "),
			wantErr: true,
			setup:   func(d testDeps) {},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				var br myerrors.BadRequestErr
				require.ErrorAs(t, err, &br)
			},
		},
		{
			name:    "phone taken by another user",
			phone:   strptr("+992900000000"),
			wantErr: true,
			setup: func(d testDeps) {
				d.Repo.On("GetUserByPhone", ctx, "+992900000000").
					Return(&models.User{ID: 99}, nil)
			},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				var c myerrors.ConflictErr
				require.ErrorAs(t, err, &c)
			},
		},
		{
			name:    "phone lookup fails",
			phone:   strptr("+992900000000"),
			wantErr: true,
			setup: func(d testDeps) {
				d.Repo.On("GetUserByPhone", ctx, "+992900000000").
					Return(nil, errors.New("db down"))
			},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
			},
		},
		{
			name:     "success updates trimmed full name only",
			fullName: strptr("  New Name  "),
			setup: func(d testDeps) {
				d.Repo.On("UpdateUserProfile", ctx, userID, strptr("New Name"), (*string)(nil)).
					Return(&models.User{ID: userID, FullName: strptr("New Name")}, nil)
			},
			assert: func(t *testing.T, u *models.User) {
				t.Helper()
				require.Equal(t, "New Name", *u.FullName)
			},
		},
		{
			name:  "success when phone belongs to same user",
			phone: strptr("+992900000001"),
			setup: func(d testDeps) {
				d.Repo.On("GetUserByPhone", ctx, "+992900000001").
					Return(&models.User{ID: userID}, nil)
				d.Repo.On("UpdateUserProfile", ctx, userID, (*string)(nil), strptr("+992900000001")).
					Return(&models.User{ID: userID, Phone: strptr("+992900000001")}, nil)
			},
			assert: func(t *testing.T, u *models.User) {
				t.Helper()
				require.Equal(t, "+992900000001", *u.Phone)
			},
		},
		{
			name:     "success with new phone and full name",
			fullName: strptr("Jane"),
			phone:    strptr("+992900000002"),
			setup: func(d testDeps) {
				d.Repo.On("GetUserByPhone", ctx, "+992900000002").
					Return(nil, fmt.Errorf("%w", myerrors.ErrNotFound))
				d.Repo.On("UpdateUserProfile", ctx, userID, strptr("Jane"), strptr("+992900000002")).
					Return(&models.User{ID: userID, FullName: strptr("Jane"), Phone: strptr("+992900000002")}, nil)
			},
			assert: func(t *testing.T, u *models.User) {
				t.Helper()
				require.Equal(t, "Jane", *u.FullName)
				require.Equal(t, "+992900000002", *u.Phone)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newTestService(t)
			tt.setup(d)

			user, err := svc.UpdateProfile(ctx, userID, tt.fullName, tt.phone)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, user)
				if tt.assertErr != nil {
					tt.assertErr(t, err)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, user)
			if tt.assert != nil {
				tt.assert(t, user)
			}
		})
	}
}
