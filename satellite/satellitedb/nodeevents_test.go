// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package satellitedb_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"common/testcontext"
	"common/uuid"
	"storx/private/teststorx"
	"storx/satellite"
	"storx/satellite/nodeevents"
	"storx/satellite/satellitedb/satellitedbtest"
)

func TestNodeEvents(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		testID := teststorx.NodeIDFromString("test")
		testEmail := "test@storx.test"
		eventType := nodeevents.Disqualified

		neFromInsert, err := db.NodeEvents().Insert(ctx, testEmail, testID, eventType)
		require.NoError(t, err)
		require.NotNil(t, neFromInsert.ID)
		require.Equal(t, testID, neFromInsert.NodeID)
		require.Equal(t, testEmail, neFromInsert.Email)
		require.Equal(t, eventType, neFromInsert.Event)
		require.NotNil(t, neFromInsert.CreatedAt)
		require.Nil(t, neFromInsert.EmailSent)

		neFromGet, err := db.NodeEvents().GetLatestByEmailAndEvent(ctx, neFromInsert.Email, neFromInsert.Event)
		require.NoError(t, err)
		require.Equal(t, neFromInsert, neFromGet)
	})
}

func TestNodeEventsUpdateEmailSent(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		testID1 := teststorx.NodeIDFromString("test1")
		testID2 := teststorx.NodeIDFromString("test2")
		testEmail1 := "test1@storx.test"
		eventType := nodeevents.Disqualified

		// Insert into node events
		event1, err := db.NodeEvents().Insert(ctx, testEmail1, testID1, eventType)
		require.NoError(t, err)

		event2, err := db.NodeEvents().Insert(ctx, testEmail1, testID2, eventType)
		require.NoError(t, err)

		// GetNextBatch should get them.
		events, err := db.NodeEvents().GetNextBatch(ctx, time.Now())
		require.NoError(t, err)

		var foundEvent1, foundEvent2 bool
		for _, ne := range events {
			switch ne.NodeID {
			case event1.NodeID:
				foundEvent1 = true
			case event2.NodeID:
				foundEvent2 = true
			default:
			}
		}
		require.True(t, foundEvent1)
		require.True(t, foundEvent2)

		// Update email sent
		require.NoError(t, db.NodeEvents().UpdateEmailSent(ctx, []uuid.UUID{event1.ID, event2.ID}, time.Now()))

		// They shouldn't be found since email_sent should have been updated.
		// It's an indirect way of checking. Not the best. We would need to add a new Read method
		// to get specific rows by ID.
		events, err = db.NodeEvents().GetNextBatch(ctx, time.Now())
		require.NoError(t, err)
		require.Len(t, events, 0)
	})
}

func TestNodeEventsUpdateLastAttempted(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		testID1 := teststorx.NodeIDFromString("test1")
		testID2 := teststorx.NodeIDFromString("test2")
		testEmail1 := "test1@storx.test"
		eventType := nodeevents.Disqualified

		event1, err := db.NodeEvents().Insert(ctx, testEmail1, testID1, eventType)
		require.NoError(t, err)

		event1, err = db.NodeEvents().GetByID(ctx, event1.ID)
		require.NoError(t, err)
		require.Nil(t, event1.LastAttempted)

		event2, err := db.NodeEvents().Insert(ctx, testEmail1, testID2, eventType)
		require.NoError(t, err)

		event2, err = db.NodeEvents().GetByID(ctx, event2.ID)
		require.NoError(t, err)
		require.Nil(t, event2.LastAttempted)

		require.NoError(t, db.NodeEvents().UpdateLastAttempted(ctx, []uuid.UUID{event1.ID, event2.ID}, time.Now()))

		event1, err = db.NodeEvents().GetByID(ctx, event1.ID)
		require.NoError(t, err)
		require.NotNil(t, event1.LastAttempted)

		event2, err = db.NodeEvents().GetByID(ctx, event2.ID)
		require.NoError(t, err)
		require.NotNil(t, event2.LastAttempted)
	})
}

func TestNodeEventsGetByID(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		testID1 := teststorx.NodeIDFromString("test1")
		testID2 := teststorx.NodeIDFromString("test2")
		testEmail1 := "test1@storx.test"
		testEmail2 := "test2@storx.test"

		eventType := nodeevents.Disqualified

		event1, err := db.NodeEvents().Insert(ctx, testEmail1, testID1, eventType)
		require.NoError(t, err)

		event2, err := db.NodeEvents().Insert(ctx, testEmail2, testID2, eventType)
		require.NoError(t, err)

		res, err := db.NodeEvents().GetByID(ctx, event1.ID)
		require.NoError(t, err)
		require.Equal(t, event1.Email, res.Email)
		require.Equal(t, event1.CreatedAt, res.CreatedAt)
		require.Equal(t, event1.Event, res.Event)

		res, err = db.NodeEvents().GetByID(ctx, event2.ID)
		require.NoError(t, err)
		require.Equal(t, event2.Email, res.Email)
		require.Equal(t, event2.CreatedAt, res.CreatedAt)
		require.Equal(t, event2.Event, res.Event)
	})
}

func TestNodeEventsGetNextBatch(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		testID1 := teststorx.NodeIDFromString("test1")
		testID2 := teststorx.NodeIDFromString("test2")
		testEmail1 := "test1@storx.test"
		testEmail2 := "test2@storx.test"

		eventType := nodeevents.Disqualified

		event1, err := db.NodeEvents().Insert(ctx, testEmail1, testID1, eventType)
		require.NoError(t, err)

		// insert one event with same email and event type, but with different node ID. It should be selected.
		event2, err := db.NodeEvents().Insert(ctx, testEmail1, testID2, eventType)
		require.NoError(t, err)

		// insert one event with same email and event type, but email_sent is not null. Should not be selected.
		event3, err := db.NodeEvents().Insert(ctx, testEmail1, testID1, eventType)
		require.NoError(t, err)

		require.NoError(t, db.NodeEvents().UpdateEmailSent(ctx, []uuid.UUID{event3.ID}, time.Now()))

		// insert one event with same email, but different type. Should not be selected.
		_, err = db.NodeEvents().Insert(ctx, testEmail1, testID1, nodeevents.BelowMinVersion)
		require.NoError(t, err)

		// insert one event with same event type, but different email. Should not be selected.
		_, err = db.NodeEvents().Insert(ctx, testEmail2, testID1, eventType)
		require.NoError(t, err)

		batch, err := db.NodeEvents().GetNextBatch(ctx, time.Now())
		require.NoError(t, err)
		require.Len(t, batch, 2)

		var foundEvent1, foundEvent2 bool
		for _, ne := range batch {
			switch ne.NodeID {
			case event1.NodeID:
				foundEvent1 = true
			case event2.NodeID:
				foundEvent2 = true
			default:
			}
		}
		require.True(t, foundEvent1)
		require.True(t, foundEvent2)
	})
}

func TestNodeEventsGetNextBatchSelectionOrder(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		id0 := teststorx.NodeIDFromString("test0")
		id1 := teststorx.NodeIDFromString("test1")
		id2 := teststorx.NodeIDFromString("test2")
		id3 := teststorx.NodeIDFromString("test3")

		email0 := "test0@storx.test"
		email1 := "test1@storx.test"
		email2 := "test2@storx.test"
		email3 := "test3@storx.test"

		eventType := nodeevents.Disqualified

		// GetNextBatch orders by last_attempted, created_at asc
		// expected selection order:
		// 1. insert0: last_attempted = nil, created_at = earliest
		// 2. insert3: last_attempted = nil, created_at = 3rd earliest
		// 3. insert2: last_attempted != nil, created_at = 4th earliest
		// 4. insert1: last_attempted later than insert2, created_at = 2nd earliest
		expectedOrder := []string{
			email0, email3, email2, email1,
		}

		_, err := db.NodeEvents().Insert(ctx, email0, id0, eventType)
		require.NoError(t, err)

		insert1, err := db.NodeEvents().Insert(ctx, email1, id1, eventType)
		require.NoError(t, err)
		require.NoError(t, err)
		require.NoError(t, db.NodeEvents().UpdateLastAttempted(ctx, []uuid.UUID{insert1.ID}, time.Now().Add(5*time.Minute)))

		insert2, err := db.NodeEvents().Insert(ctx, email2, id2, eventType)
		require.NoError(t, err)
		require.NoError(t, err)
		require.NoError(t, db.NodeEvents().UpdateLastAttempted(ctx, []uuid.UUID{insert2.ID}, time.Now()))

		_, err = db.NodeEvents().Insert(ctx, email3, id3, eventType)
		require.NoError(t, err)

		for i := 0; i < 4; i++ {
			e, err := db.NodeEvents().GetNextBatch(ctx, time.Now())
			require.NoError(t, err)
			require.Equal(t, expectedOrder[i], e[0].Email)

			require.NoError(t, db.NodeEvents().UpdateEmailSent(ctx, []uuid.UUID{e[0].ID}, time.Now()))
		}
	})
}
