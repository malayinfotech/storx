// Copyright (C) 2023 Storx Labs, Inc.
// See LICENSE for copying information.

package checker_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"common/memory"
	"common/storx"
	"common/testcontext"
	"common/testrand"
	"common/uuid"
	"storx/private/testplanet"
	"storx/satellite"
	"storx/satellite/metabase"
	"storx/satellite/metabase/rangedloop"
	"storx/satellite/repair/checker"
	"storx/satellite/repair/queue"
)

func TestIdentifyInjuredSegmentsObserver(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 4, UplinkCount: 1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.Repairer.UseRangedLoop = true
				config.RangedLoop.Parallelism = 4
				config.RangedLoop.BatchSize = 4
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		repairQueue := planet.Satellites[0].DB.RepairQueue()

		rs := storx.RedundancyScheme{
			RequiredShares: 2,
			RepairShares:   3,
			OptimalShares:  4,
			TotalShares:    5,
			ShareSize:      256,
		}

		projectID := planet.Uplinks[0].Projects[0].ID
		err := planet.Uplinks[0].CreateBucket(ctx, planet.Satellites[0], "test-bucket")
		require.NoError(t, err)

		expectedLocation := metabase.SegmentLocation{
			ProjectID:  projectID,
			BucketName: "test-bucket",
		}

		// add some valid pointers
		for x := 0; x < 10; x++ {
			expectedLocation.ObjectKey = metabase.ObjectKey(fmt.Sprintf("a-%d", x))
			insertSegment(ctx, t, planet, rs, expectedLocation, createPieces(planet, rs), nil)
		}

		// add pointer that needs repair
		expectedLocation.ObjectKey = metabase.ObjectKey("b-0")
		b0StreamID := insertSegment(ctx, t, planet, rs, expectedLocation, createLostPieces(planet, rs), nil)

		// add pointer that is unhealthy, but is expired
		expectedLocation.ObjectKey = metabase.ObjectKey("b-1")
		expiresAt := time.Now().Add(-time.Hour)
		insertSegment(ctx, t, planet, rs, expectedLocation, createLostPieces(planet, rs), &expiresAt)

		// add some valid pointers
		for x := 0; x < 10; x++ {
			expectedLocation.ObjectKey = metabase.ObjectKey(fmt.Sprintf("c-%d", x))
			insertSegment(ctx, t, planet, rs, expectedLocation, createPieces(planet, rs), nil)
		}

		_, err = planet.Satellites[0].RangedLoop.RangedLoop.Service.RunOnce(ctx)
		require.NoError(t, err)

		// check that the unhealthy, non-expired segment was added to the queue
		// and that the expired segment was ignored
		injuredSegment, err := repairQueue.Select(ctx)
		require.NoError(t, err)
		err = repairQueue.Delete(ctx, injuredSegment)
		require.NoError(t, err)

		require.Equal(t, b0StreamID, injuredSegment.StreamID)

		_, err = repairQueue.Select(ctx)
		require.Error(t, err)
	})
}

func TestIdentifyIrreparableSegmentsObserver(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 3, UplinkCount: 1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.Repairer.UseRangedLoop = true
				config.RangedLoop.Parallelism = 4
				config.RangedLoop.BatchSize = 4
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		rangeLoopService := planet.Satellites[0].RangedLoop.RangedLoop.Service

		const numberOfNodes = 10
		pieces := make(metabase.Pieces, 0, numberOfNodes)
		// use online nodes
		for i, storagenode := range planet.StorageNodes {
			pieces = append(pieces, metabase.Piece{
				Number:      uint16(i),
				StorageNode: storagenode.ID(),
			})
		}

		// simulate offline nodes
		expectedLostPieces := make(map[int32]bool)
		for i := len(pieces); i < numberOfNodes; i++ {
			pieces = append(pieces, metabase.Piece{
				Number:      uint16(i),
				StorageNode: storx.NodeID{byte(i)},
			})
			expectedLostPieces[int32(i)] = true
		}

		rs := storx.RedundancyScheme{
			ShareSize:      256,
			RequiredShares: 4,
			RepairShares:   8,
			OptimalShares:  9,
			TotalShares:    10,
		}

		projectID := planet.Uplinks[0].Projects[0].ID
		err := planet.Uplinks[0].CreateBucket(ctx, planet.Satellites[0], "test-bucket")
		require.NoError(t, err)

		expectedLocation := metabase.SegmentLocation{
			ProjectID:  projectID,
			BucketName: "test-bucket",
		}

		// when number of healthy piece is less than minimum required number of piece in redundancy,
		// the piece is considered irreparable but also will be put into repair queue

		expectedLocation.ObjectKey = "piece"
		insertSegment(ctx, t, planet, rs, expectedLocation, pieces, nil)

		expectedLocation.ObjectKey = "piece-expired"
		expiresAt := time.Now().Add(-time.Hour)
		insertSegment(ctx, t, planet, rs, expectedLocation, pieces, &expiresAt)

		_, err = rangeLoopService.RunOnce(ctx)
		require.NoError(t, err)

		// check that single irreparable segment was added repair queue
		repairQueue := planet.Satellites[0].DB.RepairQueue()
		_, err = repairQueue.Select(ctx)
		require.NoError(t, err)
		count, err := repairQueue.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		// check irreparable once again but wait a second
		time.Sleep(1 * time.Second)
		_, err = rangeLoopService.RunOnce(ctx)
		require.NoError(t, err)

		expectedLocation.ObjectKey = "piece"
		_, err = planet.Satellites[0].Metabase.DB.DeleteObjectExactVersion(ctx, metabase.DeleteObjectExactVersion{
			ObjectLocation: expectedLocation.Object(),
			Version:        metabase.DefaultVersion,
		})
		require.NoError(t, err)

		_, err = rangeLoopService.RunOnce(ctx)
		require.NoError(t, err)

		count, err = repairQueue.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})
}

func TestIgnoringCopiedSegmentsObserver(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 4, UplinkCount: 1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.Repairer.UseRangedLoop = true
				config.RangedLoop.Parallelism = 4
				config.RangedLoop.BatchSize = 4
				config.Metainfo.RS.Min = 2
				config.Metainfo.RS.Repair = 3
				config.Metainfo.RS.Success = 4
				config.Metainfo.RS.Total = 4
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		satellite := planet.Satellites[0]
		uplink := planet.Uplinks[0]
		metabaseDB := satellite.Metabase.DB

		rangedLoopService := planet.Satellites[0].RangedLoop.RangedLoop.Service
		repairQueue := satellite.DB.RepairQueue()

		err := uplink.CreateBucket(ctx, satellite, "test-bucket")
		require.NoError(t, err)

		testData := testrand.Bytes(8 * memory.KiB)
		err = uplink.Upload(ctx, satellite, "testbucket", "test/path", testData)
		require.NoError(t, err)

		project, err := uplink.OpenProject(ctx, satellite)
		require.NoError(t, err)
		defer ctx.Check(project.Close)

		segments, err := metabaseDB.TestingAllSegments(ctx)
		require.NoError(t, err)
		require.Len(t, segments, 1)

		_, err = project.CopyObject(ctx, "testbucket", "test/path", "testbucket", "empty", nil)
		require.NoError(t, err)

		segmentsAfterCopy, err := metabaseDB.TestingAllSegments(ctx)
		require.NoError(t, err)
		require.Len(t, segmentsAfterCopy, 2)

		err = planet.StopNodeAndUpdate(ctx, planet.FindNode(segments[0].Pieces[0].StorageNode))
		require.NoError(t, err)

		_, err = rangedLoopService.RunOnce(ctx)
		require.NoError(t, err)

		// check that injured segment in repair queue streamID is same that in original segment.
		injuredSegment, err := repairQueue.Select(ctx)
		require.NoError(t, err)
		require.Equal(t, segments[0].StreamID, injuredSegment.StreamID)

		// check that repair queue has only original segment, and not copied one.
		injuredSegments, err := repairQueue.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, injuredSegments)
	})
}

func TestCleanRepairQueueObserver(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 4, UplinkCount: 1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.Repairer.UseRangedLoop = true
				config.RangedLoop.Parallelism = 4
				config.RangedLoop.BatchSize = 4
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		rangedLoopService := planet.Satellites[0].RangedLoop.RangedLoop.Service
		repairQueue := planet.Satellites[0].DB.RepairQueue()
		observer := planet.Satellites[0].RangedLoop.Repair.Observer.(*checker.RangedLoopObserver)
		planet.Satellites[0].Repair.Repairer.Loop.Pause()

		rs := storx.RedundancyScheme{
			RequiredShares: 2,
			RepairShares:   3,
			OptimalShares:  4,
			TotalShares:    5,
			ShareSize:      256,
		}

		projectID := planet.Uplinks[0].Projects[0].ID
		err := planet.Uplinks[0].CreateBucket(ctx, planet.Satellites[0], "test-bucket")
		require.NoError(t, err)

		expectedLocation := metabase.SegmentLocation{
			ProjectID:  projectID,
			BucketName: "test-bucket",
		}

		healthyCount := 5
		for i := 0; i < healthyCount; i++ {
			expectedLocation.ObjectKey = metabase.ObjectKey(fmt.Sprintf("healthy-%d", i))
			insertSegment(ctx, t, planet, rs, expectedLocation, createPieces(planet, rs), nil)
		}
		unhealthyCount := 5
		unhealthyIDs := make(map[uuid.UUID]struct{})
		for i := 0; i < unhealthyCount; i++ {
			expectedLocation.ObjectKey = metabase.ObjectKey(fmt.Sprintf("unhealthy-%d", i))
			unhealthyStreamID := insertSegment(ctx, t, planet, rs, expectedLocation, createLostPieces(planet, rs), nil)
			unhealthyIDs[unhealthyStreamID] = struct{}{}
		}

		// suspend enough nodes to make healthy pointers unhealthy
		for i := rs.RequiredShares; i < rs.OptimalShares; i++ {
			require.NoError(t, planet.Satellites[0].Overlay.DB.TestSuspendNodeUnknownAudit(ctx, planet.StorageNodes[i].ID(), time.Now()))
		}

		require.NoError(t, observer.RefreshReliabilityCache(ctx))

		// check that repair queue is empty to avoid false positive
		count, err := repairQueue.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, count)

		_, err = rangedLoopService.RunOnce(ctx)
		require.NoError(t, err)

		// check that the pointers were put into the repair queue
		// and not cleaned up at the end of the checker iteration
		count, err = repairQueue.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, healthyCount+unhealthyCount, count)

		// unsuspend nodes to make the previously healthy pointers healthy again
		for i := rs.RequiredShares; i < rs.OptimalShares; i++ {
			require.NoError(t, planet.Satellites[0].Overlay.DB.TestUnsuspendNodeUnknownAudit(ctx, planet.StorageNodes[i].ID()))
		}

		require.NoError(t, observer.RefreshReliabilityCache(ctx))

		// The checker will not insert/update the now healthy segments causing
		// them to be removed from the queue at the end of the checker iteration
		_, err = rangedLoopService.RunOnce(ctx)
		require.NoError(t, err)

		// only unhealthy segments should remain
		count, err = repairQueue.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, unhealthyCount, count)

		segs, err := repairQueue.SelectN(ctx, count)
		require.NoError(t, err)
		require.Equal(t, len(unhealthyIDs), len(segs))

		for _, s := range segs {
			_, ok := unhealthyIDs[s.StreamID]
			require.True(t, ok)
		}
	})
}

func TestRepairObserver(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 4, UplinkCount: 1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.Repairer.UseRangedLoop = true
				config.RangedLoop.Parallelism = 4
				config.RangedLoop.BatchSize = 4
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		rs := storx.RedundancyScheme{
			RequiredShares: 2,
			RepairShares:   3,
			OptimalShares:  4,
			TotalShares:    5,
			ShareSize:      256,
		}

		err := planet.Uplinks[0].CreateBucket(ctx, planet.Satellites[0], "test-bucket")
		require.NoError(t, err)

		expectedLocation := metabase.SegmentLocation{
			ProjectID:  planet.Uplinks[0].Projects[0].ID,
			BucketName: "test-bucket",
		}

		// add some valid pointers
		for x := 0; x < 20; x++ {
			expectedLocation.ObjectKey = metabase.ObjectKey(fmt.Sprintf("a-%d", x))
			insertSegment(ctx, t, planet, rs, expectedLocation, createPieces(planet, rs), nil)
		}

		var injuredSegmentStreamIDs []uuid.UUID

		// add pointer that needs repair
		for x := 0; x < 5; x++ {
			expectedLocation.ObjectKey = metabase.ObjectKey(fmt.Sprintf("b-%d", x))
			injuredSegmentStreamID := insertSegment(ctx, t, planet, rs, expectedLocation, createLostPieces(planet, rs), nil)
			injuredSegmentStreamIDs = append(injuredSegmentStreamIDs, injuredSegmentStreamID)
		}

		// add pointer that is unhealthy, but is expired
		expectedLocation.ObjectKey = metabase.ObjectKey("d-1")
		expiresAt := time.Now().Add(-time.Hour)
		insertSegment(ctx, t, planet, rs, expectedLocation, createLostPieces(planet, rs), &expiresAt)

		// add some valid pointers
		for x := 0; x < 20; x++ {
			expectedLocation.ObjectKey = metabase.ObjectKey(fmt.Sprintf("c-%d", x))
			insertSegment(ctx, t, planet, rs, expectedLocation, createPieces(planet, rs), nil)
		}

		compare := func(insertedSegmentsIDs []uuid.UUID, fromRepairQueue []queue.InjuredSegment) bool {
			var repairQueueIDs []uuid.UUID
			for _, v := range fromRepairQueue {
				repairQueueIDs = append(repairQueueIDs, v.StreamID)
			}

			sort.Slice(insertedSegmentsIDs, func(i, j int) bool {
				return insertedSegmentsIDs[i].Less(insertedSegmentsIDs[j])
			})
			sort.Slice(repairQueueIDs, func(i, j int) bool {
				return repairQueueIDs[i].Less(repairQueueIDs[j])
			})

			return reflect.DeepEqual(insertedSegmentsIDs, repairQueueIDs)
		}

		type TestCase struct {
			BatchSize   int
			Parallelism int
		}

		_, err = planet.Satellites[0].RangedLoop.RangedLoop.Service.RunOnce(ctx)
		require.NoError(t, err)

		injuredSegments, err := planet.Satellites[0].DB.RepairQueue().SelectN(ctx, 10)
		require.NoError(t, err)
		require.Len(t, injuredSegments, 5)
		require.True(t, compare(injuredSegmentStreamIDs, injuredSegments))

		_, err = planet.Satellites[0].DB.RepairQueue().Clean(ctx, time.Now())
		require.NoError(t, err)

		for _, tc := range []TestCase{
			{1, 1},
			{3, 1},
			{5, 1},
			{1, 3},
			{3, 3},
			{5, 3},
			{1, 5},
			{3, 5},
			{5, 5},
		} {
			observer := planet.Satellites[0].RangedLoop.Repair.Observer
			config := planet.Satellites[0].Config
			service := rangedloop.NewService(planet.Log(), rangedloop.Config{
				Parallelism: tc.Parallelism,
				BatchSize:   tc.BatchSize,
			}, rangedloop.NewMetabaseRangeSplitter(planet.Satellites[0].Metabase.DB, config.RangedLoop.AsOfSystemInterval, config.RangedLoop.BatchSize), []rangedloop.Observer{observer})

			_, err = service.RunOnce(ctx)
			require.NoError(t, err)

			injuredSegments, err = planet.Satellites[0].DB.RepairQueue().SelectN(ctx, 10)
			require.NoError(t, err)
			require.Len(t, injuredSegments, 5)
			require.True(t, compare(injuredSegmentStreamIDs, injuredSegments))

			_, err = planet.Satellites[0].DB.RepairQueue().Clean(ctx, time.Now())
			require.NoError(t, err)
		}
	})
}
