// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package queue_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"common/testcontext"
	"common/testrand"
	"storx/satellite"
	"storx/satellite/metabase"
	"storx/satellite/repair/queue"
	"storx/satellite/satellitedb/satellitedbtest"
)

func TestInsertBufferNoCallback(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		repairQueue := db.RepairQueue()
		insertBuffer := queue.NewInsertBuffer(repairQueue, 2)

		segment1 := createInjuredSegment()
		segment2 := createInjuredSegment()
		segment3 := createInjuredSegment()

		err := insertBuffer.Insert(ctx, segment1, nil)
		require.NoError(t, err)
		count, err := repairQueue.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, count)

		err = insertBuffer.Insert(ctx, segment2, nil)
		require.NoError(t, err)
		count, err = repairQueue.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, count)

		err = insertBuffer.Insert(ctx, segment1, nil)
		require.NoError(t, err)
		count, err = repairQueue.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, count)

		err = insertBuffer.Insert(ctx, segment3, nil)
		require.NoError(t, err)
		count, err = repairQueue.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, count)
	})
}

func TestInsertBufferSingleUniqueObject(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		insertBuffer := queue.NewInsertBuffer(db.RepairQueue(), 1)

		numUnique := 0
		inc := func() {
			numUnique++
		}

		segment1 := createInjuredSegment()

		err := insertBuffer.Insert(ctx, segment1, inc)
		require.NoError(t, err)
		require.Equal(t, numUnique, 1)

		err = insertBuffer.Insert(ctx, segment1, inc)
		require.NoError(t, err)
		require.Equal(t, numUnique, 1)

		err = insertBuffer.Insert(ctx, segment1, inc)
		require.NoError(t, err)
		require.Equal(t, numUnique, 1)
	})
}

func TestInsertBufferTwoUniqueObjects(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		insertBuffer := queue.NewInsertBuffer(db.RepairQueue(), 1)

		numUnique := 0
		inc := func() {
			numUnique++
		}

		segment1 := createInjuredSegment()
		segment2 := createInjuredSegment()

		err := insertBuffer.Insert(ctx, segment1, inc)
		require.NoError(t, err)
		require.Equal(t, numUnique, 1)

		err = insertBuffer.Insert(ctx, segment2, inc)
		require.NoError(t, err)
		require.Equal(t, numUnique, 2)
	})
}

func createInjuredSegment() *queue.InjuredSegment {
	return &queue.InjuredSegment{
		StreamID: testrand.UUID(),
		Position: metabase.SegmentPosition{
			Part:  uint32(testrand.Intn(1000)),
			Index: uint32(testrand.Intn(1000)),
		},
		SegmentHealth: 10,
	}
}
