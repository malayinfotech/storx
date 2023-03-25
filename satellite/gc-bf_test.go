// Copyright (C) 2023 Storx Labs, Inc.
// See LICENSE for copying information.

package satellite_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"common/testcontext"
	"storx/private/testplanet"
	"storx/satellite"
)

func TestGCBFUseRangedLoop(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.GarbageCollectionBF.RunOnce = true
				config.GarbageCollectionBF.UseRangedLoop = true
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		err := planet.Satellites[0].GCBF.Run(ctx)
		require.NoError(t, err)
	})
}

func TestGCBFUseSegmentsLoop(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.GarbageCollectionBF.RunOnce = true
				config.GarbageCollectionBF.UseRangedLoop = false
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		err := planet.Satellites[0].GCBF.Run(ctx)
		require.NoError(t, err)
	})
}
