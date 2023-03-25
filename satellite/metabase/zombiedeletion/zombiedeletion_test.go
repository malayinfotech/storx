// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

package zombiedeletion_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"common/memory"
	"common/testcontext"
	"common/testrand"
	"storx/private/testplanet"
	"storx/satellite"
	"storx/satellite/metabase"
	"uplink/private/testuplink"
)

func TestZombieDeletion(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 1, UplinkCount: 1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.ZombieDeletion.Enabled = true
				config.ZombieDeletion.Interval = 500 * time.Millisecond
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		upl := planet.Uplinks[0]
		zombieChore := planet.Satellites[0].Core.ZombieDeletion.Chore

		zombieChore.Loop.Pause()

		err := upl.CreateBucket(ctx, planet.Satellites[0], "testbucket1")
		require.NoError(t, err)

		err = upl.CreateBucket(ctx, planet.Satellites[0], "testbucket2")
		require.NoError(t, err)

		err = upl.CreateBucket(ctx, planet.Satellites[0], "testbucket3")
		require.NoError(t, err)

		// upload regular object, will be NOT deleted
		err = upl.Upload(ctx, planet.Satellites[0], "testbucket1", "committed_object", testrand.Bytes(1*memory.KiB))
		require.NoError(t, err)

		project, err := upl.OpenProject(ctx, planet.Satellites[0])
		require.NoError(t, err)
		defer ctx.Check(project.Close)

		_, err = project.BeginUpload(ctx, "testbucket2", "zombie_object_multipart_no_segment", nil)
		require.NoError(t, err)

		// upload pending object with multipart upload, will be deleted
		info, err := project.BeginUpload(ctx, "testbucket3", "zombie_object_multipart", nil)
		require.NoError(t, err)
		partUpload, err := project.UploadPart(ctx, "testbucket3", "zombie_object_multipart", info.UploadID, 1)
		require.NoError(t, err)
		_, err = partUpload.Write(testrand.Bytes(1 * memory.KiB))
		require.NoError(t, err)
		err = partUpload.Commit()
		require.NoError(t, err)

		// Verify that all objects are in the metabase
		objects, err := planet.Satellites[0].Metabase.DB.TestingAllObjects(ctx)
		require.NoError(t, err)
		require.Len(t, objects, 3)

		// Trigger the next iteration of cleanup and wait to finish
		zombieChore.TestingSetNow(func() time.Time {
			// Set the Now function to return time after the objects zombie deadline time
			return time.Now().Add(25 * time.Hour)
		})
		zombieChore.Loop.TriggerWait()

		// Verify that only one object remain in the metabase
		objects, err = planet.Satellites[0].Metabase.DB.TestingAllObjects(ctx)
		require.NoError(t, err)
		require.Len(t, objects, 1)
		require.Equal(t, metabase.Committed, objects[0].Status)
	})
}

func TestZombieDeletion_LastSegmentActive(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 4, UplinkCount: 1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.ZombieDeletion.Enabled = true
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		// Additional test case where user is uploading 3 segments
		// each within 23h interval and when zombie deletion process
		// is executed then nothing is deleted because last segment
		// was uploaded in less then 24h.

		upl := planet.Uplinks[0]
		zombieChore := planet.Satellites[0].Core.ZombieDeletion.Chore

		zombieChore.Loop.Pause()

		err := upl.CreateBucket(ctx, planet.Satellites[0], "testbucket1")
		require.NoError(t, err)

		now := time.Now()
		// we need pending object with 3 segments uploaded
		newCtx := testuplink.WithMaxSegmentSize(ctx, 50*memory.KiB)
		project, err := upl.OpenProject(newCtx, planet.Satellites[0])
		require.NoError(t, err)
		defer ctx.Check(project.Close)

		info, err := project.BeginUpload(newCtx, "testbucket1", "pending_object", nil)
		require.NoError(t, err)

		partUpload, err := project.UploadPart(newCtx, "testbucket1", "pending_object", info.UploadID, 0)
		require.NoError(t, err)
		_, err = partUpload.Write(testrand.Bytes(140 * memory.KiB))
		require.NoError(t, err)
		err = partUpload.Commit()
		require.NoError(t, err)

		objects, err := planet.Satellites[0].Metabase.DB.TestingAllObjects(ctx)
		require.NoError(t, err)
		require.Len(t, objects, 1)
		require.Equal(t, metabase.Pending, objects[0].Status)

		segments, err := planet.Satellites[0].Metabase.DB.TestingAllSegments(ctx)
		require.NoError(t, err)
		require.Len(t, segments, 3)

		// now we need to change creation dates for all segments
		db := planet.Satellites[0].Metabase.DB.UnderlyingTagSQL()

		// change object zombie_deletion_deadline to trigger full segments verification before deletion
		zombieDeletionDeadline := now.Add(-12 * time.Hour)
		objects[0].ZombieDeletionDeadline = &zombieDeletionDeadline
		_, err = db.Exec(ctx, "UPDATE objects SET zombie_deletion_deadline = $1", zombieDeletionDeadline)
		require.NoError(t, err)

		s0CreatedAt := now.Add(3 * -23 * time.Hour)
		segments[0].CreatedAt = s0CreatedAt
		_, err = db.Exec(ctx, "UPDATE segments SET created_at = $1 WHERE stream_id = $2 AND position = $3", s0CreatedAt, objects[0].StreamID, 0)
		require.NoError(t, err)

		s1CreatedAt := now.Add(2 * -23 * time.Hour)
		segments[1].CreatedAt = s1CreatedAt
		_, err = db.Exec(ctx, "UPDATE segments SET created_at = $1 WHERE stream_id = $2 AND position = $3", s1CreatedAt, objects[0].StreamID, 1)
		require.NoError(t, err)

		// last segment should mark this object as active as it was uploaded 23h ago
		s2CreatedAt := now.Add(-23 * time.Hour)
		segments[2].CreatedAt = s2CreatedAt
		_, err = db.Exec(ctx, "UPDATE segments SET created_at = $1 WHERE stream_id = $2 AND position = $3", s2CreatedAt, objects[0].StreamID, 2)
		require.NoError(t, err)

		// running deletion process
		zombieChore.Loop.TriggerWait()

		// no changes in DB, no segment or object was deleted as last segment was uploaded less then 24h ago
		afterObjects, err := planet.Satellites[0].Metabase.DB.TestingAllObjects(ctx)
		require.NoError(t, err)

		// Diff is used because DB manipulation changes value time zone and require.Equal
		// fails on that even when value is correct
		diff := cmp.Diff(objects, afterObjects,
			cmpopts.EquateApproxTime(1*time.Second))
		require.Zero(t, diff)

		afterSegments, err := planet.Satellites[0].Metabase.DB.TestingAllSegments(ctx)
		require.NoError(t, err)
		diff = cmp.Diff(segments, afterSegments,
			cmpopts.EquateApproxTime(1*time.Second))
		require.Zero(t, diff)
	})
}
