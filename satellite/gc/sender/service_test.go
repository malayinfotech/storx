// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package sender_test

import (
	"io"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"common/memory"
	"common/testcontext"
	"common/testrand"
	"storx/private/testplanet"
	"storx/satellite/gc/bloomfilter"
	"storx/satellite/overlay"
	"storx/storagenode"
	"uplink"
)

func TestSendRetainFilters(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount:   2,
		StorageNodeCount: 1,
		UplinkCount:      1,
		Reconfigure: testplanet.Reconfigure{
			StorageNode: func(index int, config *storagenode.Config) {
				// stop processing at storagenode side so it can be inspected
				config.Retain.Concurrency = 0
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		// Set satellite 1 to store bloom filters of satellite 0
		access := planet.Uplinks[0].Access[planet.Satellites[1].NodeURL().ID]
		accessString, err := access.Serialize()
		require.NoError(t, err)

		// configure sender
		gcsender := planet.Satellites[0].GarbageCollection.Sender
		gcsender.Config.AccessGrant = accessString

		// upload 1 piece
		upl := planet.Uplinks[0]
		testData := testrand.Bytes(8 * memory.KiB)
		err = upl.Upload(ctx, planet.Satellites[0], "testbucket", "test/path/1", testData)
		require.NoError(t, err)

		// configure filter uploader
		config := planet.Satellites[0].Config.GarbageCollectionBF
		config.AccessGrant = accessString
		config.ZipBatchSize = 2

		bloomFilterService := bloomfilter.NewService(zaptest.NewLogger(t), config, planet.Satellites[0].Overlay.DB, planet.Satellites[0].Metabase.SegmentLoop)
		err = bloomFilterService.RunOnce(ctx)
		require.NoError(t, err)

		storageNode0 := planet.StorageNodes[0]
		require.Zero(t, storageNode0.Peer.Storage2.RetainService.HowManyQueued())

		// send to storagenode
		err = gcsender.RunOnce(ctx)
		require.NoError(t, err)

		require.Equal(t, 1, storageNode0.Peer.Storage2.RetainService.HowManyQueued())

		// check that zip was moved to sent
		project, err := planet.Uplinks[0].OpenProject(ctx, planet.Satellites[1])
		require.NoError(t, err)

		download, err := project.DownloadObject(ctx, gcsender.Config.Bucket, bloomfilter.LATEST, nil)
		require.NoError(t, err)

		prefix, err := io.ReadAll(download)
		require.NoError(t, err)

		err = download.Close()
		require.NoError(t, err)

		var keys []string
		it := project.ListObjects(ctx, gcsender.Config.Bucket, &uplink.ListObjectsOptions{
			Recursive: true,
			Prefix:    "sent-" + string(prefix) + "/",
		})
		require.True(t, it.Next())
		keys = append(keys, it.Item().Key)
		require.False(t, it.Next())

		sort.Strings(keys)
		require.Regexp(t, "sent-.*/.*.zip$", keys[0])
	})
}

func TestSendRetainFiltersDisqualifedNode(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount:   1,
		StorageNodeCount: 2,
		UplinkCount:      1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: testplanet.ReconfigureRS(2, 2, 2, 2),
			StorageNode: func(index int, config *storagenode.Config) {
				// stop processing at storagenode side so it can be inspected
				config.Retain.Concurrency = 0
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		// Set satellite 1 to store bloom filters of satellite 0
		access := planet.Uplinks[0].Access[planet.Satellites[0].NodeURL().ID]
		accessString, err := access.Serialize()
		require.NoError(t, err)

		// configure sender
		gcsender := planet.Satellites[0].GarbageCollection.Sender
		gcsender.Config.AccessGrant = accessString

		// upload 1 piece
		upl := planet.Uplinks[0]
		testData := testrand.Bytes(8 * memory.KiB)
		err = upl.Upload(ctx, planet.Satellites[0], "testbucket", "test/path/1", testData)
		require.NoError(t, err)

		// configure filter uploader
		config := planet.Satellites[0].Config.GarbageCollectionBF
		config.AccessGrant = accessString
		config.ZipBatchSize = 2

		bloomFilterService := bloomfilter.NewService(zaptest.NewLogger(t), config, planet.Satellites[0].Overlay.DB, planet.Satellites[0].Metabase.SegmentLoop)
		err = bloomFilterService.RunOnce(ctx)
		require.NoError(t, err)

		storageNode0 := planet.StorageNodes[0]
		err = planet.Satellites[0].Overlay.Service.DisqualifyNode(ctx, storageNode0.ID(), overlay.DisqualificationReasonAuditFailure)
		require.NoError(t, err)

		storageNode1 := planet.StorageNodes[1]
		_, err = planet.Satellites[0].DB.OverlayCache().UpdateExitStatus(ctx, &overlay.ExitStatusRequest{
			NodeID:      storageNode1.ID(),
			ExitSuccess: true,
		})
		require.NoError(t, err)

		for _, node := range planet.StorageNodes {
			require.Zero(t, node.Peer.Storage2.RetainService.HowManyQueued())
		}

		// send to storagenodes
		require.NoError(t, gcsender.RunOnce(ctx))

		for _, node := range planet.StorageNodes {
			require.Zero(t, node.Peer.Storage2.RetainService.HowManyQueued())
		}
	})
}

func TestSendInvalidZip(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount:   2,
		StorageNodeCount: 1,
		UplinkCount:      1,
		Reconfigure: testplanet.Reconfigure{
			StorageNode: func(index int, config *storagenode.Config) {
				// stop processing at storagenode side so it can be inspected
				config.Retain.Concurrency = 0
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		// Set satellite 1 to store bloom filters of satellite 0
		access := planet.Uplinks[0].Access[planet.Satellites[1].NodeURL().ID]
		accessString, err := access.Serialize()
		require.NoError(t, err)

		// configure sender
		gcsender := planet.Satellites[0].GarbageCollection.Sender
		gcsender.Config.AccessGrant = accessString

		// update LATEST file
		prefix := time.Now().Format(time.RFC3339)
		err = planet.Uplinks[0].Upload(ctx, planet.Satellites[1], gcsender.Config.Bucket, bloomfilter.LATEST, []byte(prefix))
		require.NoError(t, err)

		// upload invalid zip file
		err = planet.Uplinks[0].Upload(ctx, planet.Satellites[1], gcsender.Config.Bucket, prefix+"/wasd.zip", []byte("wasd"))
		require.NoError(t, err)

		// send to storagenode
		err = gcsender.RunOnce(ctx)
		require.NoError(t, err)

		// check that error is stored
		project, err := planet.Uplinks[0].OpenProject(ctx, planet.Satellites[1])
		require.NoError(t, err)

		var keys []string
		it := project.ListObjects(ctx, gcsender.Config.Bucket, &uplink.ListObjectsOptions{
			Recursive: true,
			Prefix:    "error-" + prefix + "/",
		})
		require.True(t, it.Next())
		keys = append(keys, it.Item().Key)
		require.True(t, it.Next())
		keys = append(keys, it.Item().Key)
		require.False(t, it.Next())

		// first is corrupt zip file and second is error text
		sort.Strings(keys)

		require.Regexp(t, "^error-.*/wasd.zip$", keys[0])
		require.Regexp(t, "^error-.*/wasd.zip.error.txt$", keys[1])

		object, err := project.DownloadObject(ctx, gcsender.Config.Bucket, keys[1], nil)
		require.NoError(t, err)
		all, err := io.ReadAll(object)
		require.NoError(t, err)
		require.Equal(t, "zip: not a valid zip file", string(all))
	})
}
