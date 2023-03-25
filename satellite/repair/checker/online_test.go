// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package checker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"common/storx"
	"common/testcontext"
	"common/testrand"
	"storx/satellite/metabase"
	"storx/satellite/nodeevents"
	"storx/satellite/overlay"
)

func TestReliabilityCache_Concurrent(t *testing.T) {
	ctx := testcontext.New(t)
	defer ctx.Cleanup()

	overlayCache, err := overlay.NewService(zap.NewNop(), fakeOverlayDB{}, fakeNodeEvents{}, "", "", overlay.Config{
		NodeSelectionCache: overlay.UploadSelectionCacheConfig{
			Staleness: 2 * time.Nanosecond,
		},
	})
	require.NoError(t, err)
	cacheCtx, cacheCancel := context.WithCancel(ctx)
	defer cacheCancel()
	ctx.Go(func() error { return overlayCache.Run(cacheCtx) })
	defer ctx.Check(overlayCache.Close)

	cache := NewReliabilityCache(overlayCache, time.Millisecond)
	var group errgroup.Group
	for i := 0; i < 10; i++ {
		group.Go(func() error {
			for i := 0; i < 10000; i++ {
				pieces := []metabase.Piece{{StorageNode: testrand.NodeID()}}
				_, err := cache.MissingPieces(ctx, time.Now(), pieces)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	require.NoError(t, group.Wait())
}

type fakeOverlayDB struct{ overlay.DB }
type fakeNodeEvents struct{ nodeevents.DB }

func (fakeOverlayDB) Reliable(context.Context, *overlay.NodeCriteria) (storx.NodeIDList, error) {
	return storx.NodeIDList{
		testrand.NodeID(),
		testrand.NodeID(),
		testrand.NodeID(),
		testrand.NodeID(),
	}, nil
}
