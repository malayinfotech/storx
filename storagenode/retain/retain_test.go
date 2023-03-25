// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information

package retain_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"

	"common/bloomfilter"
	"common/errs2"
	"common/identity/testidentity"
	"common/memory"
	"common/pb"
	"common/signing"
	"common/storx"
	"common/testcontext"
	"common/testrand"
	"storx/storage"
	"storx/storage/filestore"
	"storx/storagenode"
	"storx/storagenode/pieces"
	"storx/storagenode/retain"
	"storx/storagenode/storagenodedb/storagenodedbtest"
)

func TestRetainPieces(t *testing.T) {
	storagenodedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db storagenode.DB) {
		store := pieces.NewStore(zaptest.NewLogger(t), db.Pieces(), db.V0PieceInfo(), db.PieceExpirationDB(), db.PieceSpaceUsedDB(), pieces.DefaultConfig)
		testStore := pieces.StoreForTest{Store: store}

		const numPieces = 100
		const numPiecesToKeep = 95
		// pieces from numPiecesToKeep + numOldPieces to numPieces will
		// have a recent timestamp and thus should not be deleted
		const numOldPieces = 5

		// for this test, we set the false positive rate very low, so we can test which pieces should be deleted with precision
		filter := bloomfilter.NewOptimal(numPieces, 0.000000001)

		pieceIDs := generateTestIDs(numPieces)

		satellite0 := testidentity.MustPregeneratedSignedIdentity(0, storx.LatestIDVersion())
		satellite1 := testidentity.MustPregeneratedSignedIdentity(2, storx.LatestIDVersion())

		uplink := testidentity.MustPregeneratedSignedIdentity(3, storx.LatestIDVersion())

		// keep pieceIDs[0 : numPiecesToKeep] (old + in filter)
		// delete pieceIDs[numPiecesToKeep : numPiecesToKeep+numOldPieces] (old + not in filter)
		// keep pieceIDs[numPiecesToKeep+numOldPieces : numPieces] (recent + not in filter)
		// add all pieces to the node pieces info DB - but only count piece ids in filter
		for index, id := range pieceIDs {
			var formatVer storage.FormatVersion
			if index%2 == 0 {
				formatVer = filestore.FormatV0
			} else {
				formatVer = filestore.FormatV1
			}

			if index < numPiecesToKeep {
				filter.Add(id)
			}

			const size = 100 * memory.B

			// Write file for all satellites
			for _, satelliteID := range []storx.NodeID{satellite0.ID, satellite1.ID} {
				now := time.Now()
				w, err := testStore.WriterForFormatVersion(ctx, satelliteID, id, formatVer, pb.PieceHashAlgorithm_SHA256)
				require.NoError(t, err)

				_, err = w.Write(testrand.Bytes(size))
				require.NoError(t, err)

				require.NoError(t, w.Commit(ctx, &pb.PieceHeader{
					CreationTime: now,
				}))

				piecehash, err := signing.SignPieceHash(ctx,
					signing.SignerFromFullIdentity(uplink),
					&pb.PieceHash{
						PieceId: id,
						Hash:    []byte{0, 2, 3, 4, 5},
					})
				require.NoError(t, err)

				if formatVer == filestore.FormatV0 {
					v0db := testStore.GetV0PieceInfoDBForTest()
					err = v0db.Add(ctx, &pieces.Info{
						SatelliteID:     satelliteID,
						PieceSize:       4,
						PieceID:         id,
						PieceCreation:   now,
						UplinkPieceHash: piecehash,
						OrderLimit:      &pb.OrderLimit{},
					})
					require.NoError(t, err)
				}
			}
		}

		retainEnabled := retain.NewService(zaptest.NewLogger(t), store, retain.Config{
			Status:      retain.Enabled,
			Concurrency: 1,
			MaxTimeSkew: 0,
		})

		retainDisabled := retain.NewService(zaptest.NewLogger(t), store, retain.Config{
			Status:      retain.Disabled,
			Concurrency: 1,
			MaxTimeSkew: 0,
		})

		retainDebug := retain.NewService(zaptest.NewLogger(t), store, retain.Config{
			Status:      retain.Debug,
			Concurrency: 1,
			MaxTimeSkew: 0,
		})

		// start the retain services
		runCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		var group errgroup.Group
		group.Go(func() error {
			return retainEnabled.Run(runCtx)
		})
		group.Go(func() error {
			return retainDisabled.Run(runCtx)
		})
		group.Go(func() error {
			return retainDebug.Run(runCtx)
		})

		// expect that disabled and debug endpoints do not delete any pieces
		req := retain.Request{
			SatelliteID:   satellite0.ID,
			CreatedBefore: time.Now(),
			Filter:        filter,
		}
		queued := retainDisabled.Queue(req)
		require.True(t, queued)
		retainDisabled.TestWaitUntilEmpty()

		queued = retainDebug.Queue(req)
		require.True(t, queued)
		retainDebug.TestWaitUntilEmpty()

		satellite1Pieces, err := getAllPieceIDs(ctx, store, satellite1.ID)
		require.NoError(t, err)
		require.Equal(t, numPieces, len(satellite1Pieces))

		satellite0Pieces, err := getAllPieceIDs(ctx, store, satellite0.ID)
		require.NoError(t, err)
		require.Equal(t, numPieces, len(satellite0Pieces))

		// expect that enabled endpoint deletes the correct pieces
		queued = retainEnabled.Queue(req)
		require.True(t, queued)
		retainEnabled.TestWaitUntilEmpty()

		// check we have deleted nothing for satellite1
		satellite1Pieces, err = getAllPieceIDs(ctx, store, satellite1.ID)
		require.NoError(t, err)
		require.Equal(t, numPieces, len(satellite1Pieces))

		// check we did not delete recent pieces or retained pieces for satellite0
		// also check that we deleted the correct pieces for satellite0
		satellite0Pieces, err = getAllPieceIDs(ctx, store, satellite0.ID)
		require.NoError(t, err)
		require.Equal(t, numPieces-numOldPieces, len(satellite0Pieces))

		for _, id := range pieceIDs[:numPiecesToKeep] {
			require.Contains(t, satellite0Pieces, id, "piece should not have been deleted (not in bloom filter)")
		}

		for _, id := range pieceIDs[numPiecesToKeep+numOldPieces:] {
			require.Contains(t, satellite0Pieces, id, "piece should not have been deleted (recent piece)")
		}

		for _, id := range pieceIDs[numPiecesToKeep : numPiecesToKeep+numOldPieces] {
			require.NotContains(t, satellite0Pieces, id, "piece should have been deleted")
		}

		// shut down retain services
		cancel()
		err = group.Wait()
		require.True(t, errs2.IsCanceled(err))
	})
}

func getAllPieceIDs(ctx context.Context, store *pieces.Store, satellite storx.NodeID) (pieceIDs []storx.PieceID, err error) {
	err = store.WalkSatellitePieces(ctx, satellite, func(pieceAccess pieces.StoredPieceAccess) error {
		pieceIDs = append(pieceIDs, pieceAccess.PieceID())
		return nil
	})
	return pieceIDs, err
}

// generateTestIDs generates n piece ids.
func generateTestIDs(n int) []storx.PieceID {
	ids := make([]storx.PieceID, n)
	for i := range ids {
		ids[i] = testrand.PieceID()
	}
	return ids
}
