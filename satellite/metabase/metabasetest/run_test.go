// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package metabasetest_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"common/testcontext"
	_ "private/dbutil/cockroachutil" // register cockroach driver
	"storx/satellite/metabase"
	"storx/satellite/metabase/metabasetest"
)

func TestSetup(t *testing.T) {
	metabasetest.Run(t, func(ctx *testcontext.Context, t *testing.T, db *metabase.DB) {
		err := db.Ping(ctx)
		require.NoError(t, err)

		_, err = db.TestingGetState(ctx)
		require.NoError(t, err)
	})
}
