// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package nodetally_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"common/encryption"
	"common/memory"
	"common/storx"
	"common/testcontext"
	"common/testrand"
	"storx/private/testplanet"
	"storx/satellite"
)

func TestSingleObjectNodeTallyRangedLoop(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 4, UplinkCount: 1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.Tally.UseRangedLoop = true
				config.RangedLoop.Parallelism = 4
				config.RangedLoop.BatchSize = 4
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		timespanHours := 2

		firstNow := time.Date(2020, 8, 8, 8, 8, 8, 0, time.UTC)
		obs := planet.Satellites[0].RangedLoop.Accounting.NodeTallyObserver
		obs.SetNow(func() time.Time {
			return firstNow
		})

		// first run to zero out the database
		_, err := planet.Satellites[0].RangedLoop.RangedLoop.Service.RunOnce(ctx)
		require.NoError(t, err)

		// Setup: create 50KiB of data for the uplink to upload
		expectedData := testrand.Bytes(50 * memory.KiB)

		// TODO uplink currently hardcode block size so we need to use the same value in test
		encryptionParameters := storx.EncryptionParameters{
			CipherSuite: storx.EncAESGCM,
			BlockSize:   29 * 256 * memory.B.Int32(),
		}
		expectedTotalBytes, err := encryption.CalcEncryptedSize(int64(len(expectedData)), encryptionParameters)
		require.NoError(t, err)

		// Execute test: upload a file, then calculate at rest data
		expectedBucketName := "testbucket"
		err = planet.Uplinks[0].Upload(ctx, planet.Satellites[0], expectedBucketName, "test/path", expectedData)
		require.NoError(t, err)

		secondNow := firstNow.Add(2 * time.Hour)
		obs.SetNow(func() time.Time {
			return secondNow
		})

		_, err = planet.Satellites[0].RangedLoop.RangedLoop.Service.RunOnce(ctx)
		require.NoError(t, err)

		// Confirm the correct number of shares were stored
		rs := satelliteRS(t, planet.Satellites[0])
		if !correctRedundencyScheme(len(obs.Node), rs) {
			t.Fatalf("expected between: %d and %d, actual: %d", rs.RepairShares, rs.TotalShares, len(obs.Node))
		}

		// Confirm the correct number of bytes were stored on each node
		for _, actualTotalBytes := range obs.Node {
			require.EqualValues(t, expectedTotalBytes, actualTotalBytes)
		}

		// Confirm that tallies where saved to DB
		tallies, err := planet.Satellites[0].DB.StoragenodeAccounting().GetTalliesSince(ctx, secondNow.Add(-1*time.Second))
		require.NoError(t, err)
		require.LessOrEqual(t, len(tallies), int(rs.TotalShares))
		require.GreaterOrEqual(t, len(tallies), int(rs.OptimalShares))

		for _, tally := range tallies {
			require.Equal(t, obs.Node[tally.NodeID]*float64(timespanHours), tally.DataTotal)
		}

		thirdNow := secondNow.Add(2 * time.Hour)
		obs.SetNow(func() time.Time {
			return thirdNow
		})

		_, err = planet.Satellites[0].RangedLoop.RangedLoop.Service.RunOnce(ctx)
		require.NoError(t, err)

		tallies, err = planet.Satellites[0].DB.StoragenodeAccounting().GetTalliesSince(ctx, thirdNow.Add(-1*time.Second))
		require.NoError(t, err)
		require.LessOrEqual(t, len(tallies), int(rs.TotalShares))
		require.GreaterOrEqual(t, len(tallies), int(rs.OptimalShares))

		for _, tally := range tallies {
			require.Equal(t, obs.Node[tally.NodeID]*float64(timespanHours), tally.DataTotal)
		}
	})
}

func TestManyObjectsNodeTallyRangedLoop(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 4, UplinkCount: 1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.Tally.UseRangedLoop = true
				config.RangedLoop.Parallelism = 4
				config.RangedLoop.BatchSize = 4
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		timespanHours := 2
		numObjects := 10

		now := time.Date(2020, 8, 8, 8, 8, 8, 0, time.UTC)
		lastTally := now.Add(time.Duration(-1 * timespanHours * int(time.Hour)))
		// Set previous accounting run timestamp
		err := planet.Satellites[0].DB.StoragenodeAccounting().DeleteTalliesBefore(ctx, now.Add(1*time.Second), 5000)
		require.NoError(t, err)
		err = planet.Satellites[0].DB.StoragenodeAccounting().SaveTallies(ctx, lastTally, map[storx.NodeID]float64{
			planet.StorageNodes[0].ID(): 0,
			planet.StorageNodes[1].ID(): 0,
			planet.StorageNodes[2].ID(): 0,
			planet.StorageNodes[3].ID(): 0,
		})
		require.NoError(t, err)

		// Setup: create 50KiB of data for the uplink to upload
		expectedData := testrand.Bytes(50 * memory.KiB)

		// TODO uplink currently hardcode block size so we need to use the same value in test
		encryptionParameters := storx.EncryptionParameters{
			CipherSuite: storx.EncAESGCM,
			BlockSize:   29 * 256 * memory.B.Int32(),
		}
		expectedBytesPerPiece, err := encryption.CalcEncryptedSize(int64(len(expectedData)), encryptionParameters)
		require.NoError(t, err)

		// Execute test: upload a file, then calculate at rest data
		expectedBucketName := "testbucket"
		for i := 0; i < numObjects; i++ {
			err = planet.Uplinks[0].Upload(ctx, planet.Satellites[0], expectedBucketName, fmt.Sprintf("test/path%d", i), expectedData)
			require.NoError(t, err)
		}

		rangedLoop := planet.Satellites[0].RangedLoop
		obs := rangedLoop.Accounting.NodeTallyObserver
		obs.SetNow(func() time.Time {
			return now
		})

		_, err = planet.Satellites[0].RangedLoop.RangedLoop.Service.RunOnce(ctx)
		require.NoError(t, err)

		rs := satelliteRS(t, planet.Satellites[0])
		minExpectedBytes := numObjects * int(expectedBytesPerPiece) * int(rs.OptimalShares)
		maxExpectedBytes := numObjects * int(expectedBytesPerPiece) * int(rs.TotalShares)

		// Confirm the correct number of bytes were stored on each node
		totalBytes := 0
		for _, actualTotalBytes := range obs.Node {
			totalBytes += int(actualTotalBytes)
		}
		require.LessOrEqual(t, totalBytes, maxExpectedBytes)
		require.GreaterOrEqual(t, totalBytes, minExpectedBytes)

		// Confirm that tallies where saved to DB
		tallies, err := planet.Satellites[0].DB.StoragenodeAccounting().GetTalliesSince(ctx, now.Add(-1*time.Second))
		require.NoError(t, err)

		totalByteHours := 0
		for _, tally := range tallies {
			totalByteHours += int(tally.DataTotal)
		}

		require.LessOrEqual(t, totalByteHours, maxExpectedBytes*timespanHours)
		require.GreaterOrEqual(t, totalByteHours, minExpectedBytes*timespanHours)
	})
}

func TestExpiredObjectsNotCountedInNodeTally(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 4, UplinkCount: 1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.Tally.UseRangedLoop = true
				config.RangedLoop.Parallelism = 1
				config.RangedLoop.BatchSize = 4
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		timespanHours := 2
		numObjects := 10

		now := time.Date(2030, 8, 8, 8, 8, 8, 0, time.UTC)
		obs := planet.Satellites[0].RangedLoop.Accounting.NodeTallyObserver
		obs.SetNow(func() time.Time {
			return now
		})

		lastTally := now.Add(time.Duration(-1 * timespanHours * int(time.Hour)))
		err := planet.Satellites[0].DB.StoragenodeAccounting().SaveTallies(ctx, lastTally, map[storx.NodeID]float64{
			planet.StorageNodes[0].ID(): 0,
			planet.StorageNodes[1].ID(): 0,
			planet.StorageNodes[2].ID(): 0,
			planet.StorageNodes[3].ID(): 0,
		})
		require.NoError(t, err)

		// Setup: create 50KiB of data for the uplink to upload
		expectedData := testrand.Bytes(50 * memory.KiB)

		// Upload expired objects and the same number of soon-to-expire objects
		expectedBucketName := "testbucket"
		for i := 0; i < numObjects; i++ {
			err = planet.Uplinks[0].UploadWithExpiration(
				ctx, planet.Satellites[0], expectedBucketName, fmt.Sprint("test/pathA", i), expectedData, now.Add(-1*time.Second),
			)
			require.NoError(t, err)
			err = planet.Uplinks[0].UploadWithExpiration(
				ctx, planet.Satellites[0], expectedBucketName, fmt.Sprint("test/pathB", i), expectedData, now.Add(1*time.Second),
			)
			require.NoError(t, err)
		}

		_, err = planet.Satellites[0].RangedLoop.RangedLoop.Service.RunOnce(ctx)
		require.NoError(t, err)

		rs := satelliteRS(t, planet.Satellites[0])
		// TODO uplink currently hardcode block size so we need to use the same value in test
		encryptionParameters := storx.EncryptionParameters{
			CipherSuite: storx.EncAESGCM,
			BlockSize:   29 * 256 * memory.B.Int32(),
		}
		expectedBytesPerPiece, err := encryption.CalcEncryptedSize(int64(len(expectedData)), encryptionParameters)
		require.NoError(t, err)
		minExpectedBytes := numObjects * int(expectedBytesPerPiece) * int(rs.OptimalShares)
		maxExpectedBytes := numObjects * int(expectedBytesPerPiece) * int(rs.TotalShares)

		// Confirm the correct number of bytes were stored on each node
		totalBytes := 0
		for _, actualTotalBytes := range obs.Node {
			totalBytes += int(actualTotalBytes)
		}
		require.LessOrEqual(t, totalBytes, maxExpectedBytes)
		require.GreaterOrEqual(t, totalBytes, minExpectedBytes)
	})
}
