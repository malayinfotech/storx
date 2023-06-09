// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

package uitest

import (
	"os"
	"testing"

	"github.com/go-rod/rod"

	"common/testcontext"
	"storx/multinode"
	"storx/private/testplanet"
)

// Multinode starts a new UI test with multinode instance(s).
func Multinode(t *testing.T, multinodeCount int, test Test) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 4, MultinodeCount: multinodeCount,
		Reconfigure: testplanet.Reconfigure{
			Multinode: func(index int, config *multinode.Config) {
				if dir := os.Getenv("STORX_TEST_MULTINODE_WEB"); dir != "" {
					config.Console.StaticDir = dir
				}
			},
			Satellite: configureSatellite,
		},
		NonParallel: true,
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		Browser(t, ctx, planet, func(browser *rod.Browser) {
			test(t, ctx, planet, browser)
		})
	})
}
