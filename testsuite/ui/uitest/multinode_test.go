// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

package uitest_test

import (
	"testing"

	"github.com/go-rod/rod"

	"common/testcontext"
	"storx/private/testplanet"
	"storx/testsuite/ui/uitest"
)

func TestMultinode(t *testing.T) {
	uitest.Multinode(t, 1, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet, browser *rod.Browser) {
		t.Log("working")
	})
}
