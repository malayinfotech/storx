// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information

package piecestore_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeebo/errs"

	"common/identity"
	"common/memory"
	"common/pb"
	"common/signing"
	"common/storx"
	"common/testcontext"
	"common/testrand"
	"storx/private/testplanet"
)

const oneWeek = 7 * 24 * time.Hour

func TestOrderLimitPutValidation(t *testing.T) {
	for _, tt := range []struct {
		testName            string
		useUnknownSatellite bool
		pieceID             storx.PieceID
		action              pb.PieceAction
		serialNumber        storx.SerialNumber
		pieceExpiration     time.Duration
		orderExpiration     time.Duration
		limit               int64
		availableSpace      int64
		err                 string
	}{
		{
			testName:            "unapproved satellite id",
			useUnknownSatellite: true,
			pieceID:             storx.PieceID{1},
			action:              pb.PieceAction_PUT,
			serialNumber:        storx.SerialNumber{1},
			pieceExpiration:     oneWeek,
			orderExpiration:     oneWeek,
			limit:               memory.KiB.Int64(),
			err:                 " is untrusted",
		},
		{
			testName:        "approved satellite id",
			pieceID:         storx.PieceID{2},
			action:          pb.PieceAction_PUT,
			serialNumber:    storx.SerialNumber{2},
			pieceExpiration: oneWeek,
			orderExpiration: oneWeek,
			limit:           10 * memory.KiB.Int64(),
		},
		{
			testName:        "wrong action type",
			pieceID:         storx.PieceID{3},
			action:          pb.PieceAction_GET,
			serialNumber:    storx.SerialNumber{3},
			pieceExpiration: oneWeek,
			orderExpiration: oneWeek,
			limit:           memory.KiB.Int64(),
			err:             "expected put or put repair action got GET",
		},
		{
			testName:        "piece expired",
			pieceID:         storx.PieceID{4},
			action:          pb.PieceAction_PUT,
			serialNumber:    storx.SerialNumber{4},
			pieceExpiration: -4 * 24 * time.Hour,
			orderExpiration: oneWeek,
			limit:           memory.KiB.Int64(),
			err:             "piece expired:",
		},
		{
			testName:        "limit is negative",
			pieceID:         storx.PieceID{5},
			action:          pb.PieceAction_PUT,
			serialNumber:    storx.SerialNumber{5},
			pieceExpiration: oneWeek,
			orderExpiration: oneWeek,
			limit:           -1,
			err:             "order limit is negative",
		},
		{
			testName:        "order limit expired",
			pieceID:         storx.PieceID{6},
			action:          pb.PieceAction_PUT,
			serialNumber:    storx.SerialNumber{6},
			pieceExpiration: oneWeek,
			orderExpiration: -4 * 24 * time.Hour,
			limit:           memory.KiB.Int64(),
			err:             "order expired:",
		},
		{
			testName:        "allocated space limit",
			pieceID:         storx.PieceID{8},
			action:          pb.PieceAction_PUT,
			serialNumber:    storx.SerialNumber{8},
			pieceExpiration: oneWeek,
			orderExpiration: oneWeek,
			limit:           10 * memory.KiB.Int64(),
			availableSpace:  5 * memory.KiB.Int64(),
			err:             "not enough available disk space",
		},
	} {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			testplanet.Run(t, testplanet.Config{
				SatelliteCount: 1, StorageNodeCount: 1, UplinkCount: 1,
			}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {

				// set desirable space
				setSpace(ctx, t, planet, tt.availableSpace)

				client, err := planet.Uplinks[0].DialPiecestore(ctx, planet.StorageNodes[0])
				require.NoError(t, err)
				defer ctx.Check(client.Close)

				signer := signing.SignerFromFullIdentity(planet.Satellites[0].Identity)
				satellite := planet.Satellites[0].Identity
				if tt.useUnknownSatellite {
					unapprovedSatellite, err := planet.NewIdentity()
					require.NoError(t, err)
					signer = signing.SignerFromFullIdentity(unapprovedSatellite)
					satellite = unapprovedSatellite
				}

				orderLimit, piecePrivateKey := GenerateOrderLimit(
					t,
					satellite.ID,
					planet.StorageNodes[0].ID(),
					tt.pieceID,
					tt.action,
					tt.serialNumber,
					tt.pieceExpiration,
					tt.orderExpiration,
					tt.limit,
				)

				orderLimit, err = signing.SignOrderLimit(ctx, signer, orderLimit)
				require.NoError(t, err)

				buffer := make([]byte, 10*memory.KiB)
				testrand.Read(buffer)

				_, err = client.UploadReader(ctx, orderLimit, piecePrivateKey, bytes.NewReader(buffer))
				if tt.err != "" {
					require.Error(t, err)
					require.Contains(t, err.Error(), tt.err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestOrderLimitGetValidation(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 1, UplinkCount: 1,
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {

		defaultPieceSize := 10 * memory.KiB

		for _, storageNode := range planet.StorageNodes {
			err := storageNode.DB.Bandwidth().Add(ctx, planet.Satellites[0].ID(), pb.PieceAction_GET, memory.TB.Int64()-(15*memory.KiB.Int64()), time.Now())
			require.NoError(t, err)
		}

		{ // upload test piece
			client, err := planet.Uplinks[0].DialPiecestore(ctx, planet.StorageNodes[0])
			require.NoError(t, err)
			defer ctx.Check(client.Close)

			signer := signing.SignerFromFullIdentity(planet.Satellites[0].Identity)
			satellite := planet.Satellites[0].Identity

			orderLimit, piecePrivateKey := GenerateOrderLimit(
				t,
				satellite.ID,
				planet.StorageNodes[0].ID(),
				storx.PieceID{1},
				pb.PieceAction_PUT,
				storx.SerialNumber{0},
				oneWeek,
				oneWeek,
				defaultPieceSize.Int64(),
			)

			orderLimit, err = signing.SignOrderLimit(ctx, signer, orderLimit)
			require.NoError(t, err)

			_, err = client.UploadReader(ctx, orderLimit, piecePrivateKey, bytes.NewReader(testrand.Bytes(defaultPieceSize)))
			require.NoError(t, err)
		}

		// wait for all requests to finish to ensure that the upload usage has been
		// accounted for.
		require.NoError(t, planet.WaitForStorageNodeEndpoints(ctx))

		for _, tt := range []struct {
			satellite       *identity.FullIdentity
			pieceID         storx.PieceID
			action          pb.PieceAction
			serialNumber    storx.SerialNumber
			pieceExpiration time.Duration
			orderExpiration time.Duration
			limit           int64
			err             string
		}{
			{ // incorrect action - PUT rather than GET
				pieceID:         storx.PieceID{1},
				action:          pb.PieceAction_PUT,
				serialNumber:    storx.SerialNumber{1},
				pieceExpiration: oneWeek,
				orderExpiration: oneWeek,
				limit:           10 * memory.KiB.Int64(),
				err:             "expected get or get repair or audit action got PUT",
			},
		} {
			func() {
				client, err := planet.Uplinks[0].DialPiecestore(ctx, planet.StorageNodes[0])
				require.NoError(t, err)
				defer ctx.Check(client.Close)

				signer := signing.SignerFromFullIdentity(planet.Satellites[0].Identity)
				satellite := planet.Satellites[0].Identity
				if tt.satellite != nil {
					signer = signing.SignerFromFullIdentity(tt.satellite)
					satellite = tt.satellite
				}

				orderLimit, piecePrivateKey := GenerateOrderLimit(
					t,
					satellite.ID,
					planet.StorageNodes[0].ID(),
					tt.pieceID,
					tt.action,
					tt.serialNumber,
					tt.pieceExpiration,
					tt.orderExpiration,
					tt.limit,
				)

				orderLimit, err = signing.SignOrderLimit(ctx, signer, orderLimit)
				require.NoError(t, err)

				downloader, err := client.Download(ctx, orderLimit, piecePrivateKey, 0, tt.limit)
				require.NoError(t, err)

				buffer, readErr := io.ReadAll(downloader)
				closeErr := downloader.Close()
				err = errs.Combine(readErr, closeErr)
				if tt.err != "" {
					assert.Equal(t, 0, len(buffer))
					require.Error(t, err)
					require.Contains(t, err.Error(), tt.err)
				} else {
					require.NoError(t, err)
				}
			}()
		}
	})
}

func setSpace(ctx context.Context, t *testing.T, planet *testplanet.Planet, space int64) {
	if space == 0 {
		return
	}
	for _, storageNode := range planet.StorageNodes {
		availableSpace, err := storageNode.Storage2.Monitor.AvailableSpace(ctx)
		require.NoError(t, err)
		// add these bytes to the space used cache so that we can test what happens
		// when we exceeded available space on the storagenode
		err = storageNode.DB.PieceSpaceUsedDB().UpdatePieceTotals(ctx, availableSpace-space, availableSpace-space)
		require.NoError(t, err)
		err = storageNode.Storage2.CacheService.Init(ctx)
		require.NoError(t, err)
	}
}
