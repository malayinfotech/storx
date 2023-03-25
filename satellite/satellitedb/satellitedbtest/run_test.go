// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package satellitedbtest_test

import (
	"testing"

	"common/testcontext"
	"storx/satellite"
	"storx/satellite/satellitedb/satellitedbtest"
)

func TestDatabase(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
	})
}
