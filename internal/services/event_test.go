// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"shb/internal/models"
	"shb/pkg/myerrors"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_CreateEvent(t *testing.T) {
	ctx := context.Background()
	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-24 * time.Hour)

	tests := []struct {
		name      string
		event     *models.Event
		setup     func(d testDeps)
		wantID    int
		wantErr   bool
		assertErr func(t *testing.T, err error)
	}{
		{
			name:    "event date in the past",
			event:   &models.Event{InstitutionID: 5, EventDate: past},
			setup:   func(d testDeps) {},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				var br myerrors.BadRequestErr
				require.ErrorAs(t, err, &br)
			},
		},
		{
			name:  "institution not found",
			event: &models.Event{InstitutionID: 5, EventDate: future},
			setup: func(d testDeps) {
				d.Repo.On("GetInstitutionByID", ctx, 5).Return(nil, errors.New("no rows"))
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				var br myerrors.BadRequestErr
				require.ErrorAs(t, err, &br)
			},
		},
		{
			name:  "create event repo error",
			event: &models.Event{InstitutionID: 5, EventDate: future},
			setup: func(d testDeps) {
				d.Repo.On("GetInstitutionByID", ctx, 5).Return(&models.Institution{ID: 5}, nil)
				d.Repo.On("CreateEvent", ctx, mock.AnythingOfType("*models.Event")).Return(0, errors.New("insert fail"))
			},
			wantErr: true,
		},
		{
			name:  "success",
			event: &models.Event{InstitutionID: 5, EventDate: future},
			setup: func(d testDeps) {
				d.Repo.On("GetInstitutionByID", ctx, 5).Return(&models.Institution{ID: 5}, nil)
				d.Repo.On("CreateEvent", ctx, mock.AnythingOfType("*models.Event")).Return(42, nil)
			},
			wantID:  42,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newTestService(t)
			tt.setup(d)

			id, err := svc.CreateEvent(ctx, tt.event)
			if tt.wantErr {
				require.Error(t, err)
				if tt.assertErr != nil {
					tt.assertErr(t, err)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantID, id)
		})
	}
}

func TestService_GetAllEvents(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		q := models.EventListQuery{Limit: 10, Offset: 0}
		page := &models.EventPage{Total: 1, Limit: 10}
		d.Repo.On("GetAllEvents", ctx, q).Return(page, nil)

		got, err := svc.GetAllEvents(ctx, q)
		require.NoError(t, err)
		require.Equal(t, page, got)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		q := models.EventListQuery{Limit: 10}
		d.Repo.On("GetAllEvents", ctx, q).Return(nil, errors.New("db"))

		got, err := svc.GetAllEvents(ctx, q)
		require.Error(t, err)
		require.Nil(t, got)
	})
}

func TestService_GetEventByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		ev := &models.Event{ID: 7}
		d.Repo.On("GetEventByID", ctx, 7).Return(ev, nil)

		got, err := svc.GetEventByID(ctx, 7)
		require.NoError(t, err)
		require.Equal(t, ev, got)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetEventByID", ctx, 7).Return(nil, errors.New("db"))

		got, err := svc.GetEventByID(ctx, 7)
		require.Error(t, err)
		require.Nil(t, got)
	})
}

func TestService_GetEventDetail(t *testing.T) {
	ctx := context.Background()
	q := models.EventDetailQuery{EventID: 3, ViewerUserID: 1}

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		resp := &models.EventResponse{ID: 3}
		d.Repo.On("GetEventDetail", ctx, q).Return(resp, nil)

		got, err := svc.GetEventDetail(ctx, q)
		require.NoError(t, err)
		require.Equal(t, resp, got)
	})

	t.Run("not found mapped to ErrNotFound", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetEventDetail", ctx, q).Return(nil, pgx.ErrNoRows)

		got, err := svc.GetEventDetail(ctx, q)
		require.Error(t, err)
		require.Nil(t, got)
		require.ErrorIs(t, err, myerrors.ErrNotFound)
	})

	t.Run("generic error passed through", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetEventDetail", ctx, q).Return(nil, errors.New("db"))

		got, err := svc.GetEventDetail(ctx, q)
		require.Error(t, err)
		require.Nil(t, got)
		require.NotErrorIs(t, err, myerrors.ErrNotFound)
	})
}

func TestService_JoinEvent(t *testing.T) {
	ctx := context.Background()
	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-24 * time.Hour)

	t.Run("event not found", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetEventByID", ctx, 1).Return(nil, errors.New("db"))

		err := svc.JoinEvent(ctx, 1, 2)
		require.Error(t, err)
		var br myerrors.BadRequestErr
		require.ErrorAs(t, err, &br)
	})

	t.Run("cannot join past event", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetEventByID", ctx, 1).Return(&models.Event{ID: 1, EventDate: past}, nil)

		err := svc.JoinEvent(ctx, 1, 2)
		require.Error(t, err)
		var br myerrors.BadRequestErr
		require.ErrorAs(t, err, &br)
	})

	t.Run("join repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetEventByID", ctx, 1).Return(&models.Event{ID: 1, EventDate: future}, nil)
		d.Repo.On("JoinEvent", ctx, 1, 2).Return(errors.New("db"))

		err := svc.JoinEvent(ctx, 1, 2)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetEventByID", ctx, 1).Return(&models.Event{ID: 1, EventDate: future}, nil)
		d.Repo.On("JoinEvent", ctx, 1, 2).Return(nil)

		err := svc.JoinEvent(ctx, 1, 2)
		require.NoError(t, err)
	})
}

func TestService_LeaveEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("event not found", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetEventByID", ctx, 1).Return(nil, errors.New("db"))

		err := svc.LeaveEvent(ctx, 1, 2)
		require.Error(t, err)
		var br myerrors.BadRequestErr
		require.ErrorAs(t, err, &br)
	})

	t.Run("leave repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetEventByID", ctx, 1).Return(&models.Event{ID: 1}, nil)
		d.Repo.On("LeaveEvent", ctx, 1, 2).Return(errors.New("db"))

		err := svc.LeaveEvent(ctx, 1, 2)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetEventByID", ctx, 1).Return(&models.Event{ID: 1}, nil)
		d.Repo.On("LeaveEvent", ctx, 1, 2).Return(nil)

		err := svc.LeaveEvent(ctx, 1, 2)
		require.NoError(t, err)
	})
}

func TestService_GetInstitutionEvents(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		events := []*models.EventResponse{{ID: 1}, {ID: 2}}
		d.Repo.On("GetInstitutionEvents", ctx, 5).Return(events, nil)

		got, err := svc.GetInstitutionEvents(ctx, 5)
		require.NoError(t, err)
		require.Equal(t, events, got)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("GetInstitutionEvents", ctx, 5).Return(nil, errors.New("db"))

		got, err := svc.GetInstitutionEvents(ctx, 5)
		require.Error(t, err)
		require.Nil(t, got)
	})
}

func TestService_ApproveEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("UpdateEventStatus", ctx, 1, "approved").Return(nil)

		require.NoError(t, svc.ApproveEvent(ctx, 1))
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("UpdateEventStatus", ctx, 1, "approved").Return(errors.New("db"))

		require.Error(t, svc.ApproveEvent(ctx, 1))
	})
}

func TestService_RejectEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("UpdateEventStatus", ctx, 1, "rejected").Return(nil)

		require.NoError(t, svc.RejectEvent(ctx, 1))
	})

	t.Run("repo error", func(t *testing.T) {
		svc, d := newTestService(t)
		d.Repo.On("UpdateEventStatus", ctx, 1, "rejected").Return(errors.New("db"))

		require.Error(t, svc.RejectEvent(ctx, 1))
	})
}
