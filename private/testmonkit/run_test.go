// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information

package testmonkit_test

import (
	"context"
	"testing"
	"time"

	"storx/private/testmonkit"
)

func TestBasic(t *testing.T) {
	// Set STORX_TEST_MONKIT=svg,json for this to see the output.
	testmonkit.Run(context.Background(), t, func(ctx context.Context) {
		time.Sleep(100 * time.Millisecond)
	})
}
