// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package satellitedb_test

import (
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"common/testcontext"
	"common/testrand"
	"common/uuid"
	"storx/satellite"
	"storx/satellite/console"
	"storx/satellite/satellitedb/satellitedbtest"
)

func TestAccountFreezeEvents(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		randUsageLimits := func() console.UsageLimits {
			return console.UsageLimits{Storage: rand.Int63(), Bandwidth: rand.Int63(), Segment: rand.Int63()}
		}

		event := &console.AccountFreezeEvent{
			UserID: testrand.UUID(),
			Type:   console.Freeze,
			Limits: &console.AccountFreezeEventLimits{
				User: randUsageLimits(),
				Projects: map[uuid.UUID]console.UsageLimits{
					testrand.UUID(): randUsageLimits(),
					testrand.UUID(): randUsageLimits(),
				},
			},
		}

		eventsDB := db.Console().AccountFreezeEvents()

		t.Run("Can't insert nil event", func(t *testing.T) {
			_, err := eventsDB.Upsert(ctx, nil)
			require.Error(t, err)
		})

		t.Run("Insert event", func(t *testing.T) {
			dbEvent, err := eventsDB.Upsert(ctx, event)
			require.NoError(t, err)
			require.NotNil(t, dbEvent)
			require.WithinDuration(t, time.Now(), dbEvent.CreatedAt, time.Minute)
			dbEvent.CreatedAt = event.CreatedAt
			require.Equal(t, event, dbEvent)
		})

		t.Run("Get event", func(t *testing.T) {
			dbEvent, err := eventsDB.Get(ctx, event.UserID, event.Type)
			require.NoError(t, err)
			require.NotNil(t, dbEvent)
			dbEvent.CreatedAt = event.CreatedAt
			require.Equal(t, event, dbEvent)
		})

		t.Run("Update event limits", func(t *testing.T) {
			event.Limits = &console.AccountFreezeEventLimits{
				User: randUsageLimits(),
				Projects: map[uuid.UUID]console.UsageLimits{
					testrand.UUID(): randUsageLimits(),
				},
			}

			_, err := eventsDB.Upsert(ctx, event)
			require.NoError(t, err)
			dbEvent, err := eventsDB.Get(ctx, event.UserID, event.Type)
			require.NoError(t, err)
			require.Equal(t, event.Limits, dbEvent.Limits)

			event.Limits = nil
			_, err = eventsDB.Upsert(ctx, event)
			require.NoError(t, err)
			dbEvent, err = eventsDB.Get(ctx, event.UserID, event.Type)
			require.NoError(t, err)
			require.Nil(t, dbEvent.Limits)
		})

		t.Run("Delete event", func(t *testing.T) {
			require.NoError(t, eventsDB.DeleteAllByUserID(ctx, event.UserID))
			_, err := eventsDB.Get(ctx, event.UserID, event.Type)
			require.ErrorIs(t, err, sql.ErrNoRows)
		})
	})
}
