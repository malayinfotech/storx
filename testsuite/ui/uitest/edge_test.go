// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

package uitest_test

import (
	"testing"

	"github.com/go-rod/rod"

	"common/testcontext"
	"storx/testsuite/ui/uitest"
)

func TestEdge(t *testing.T) {
	uitest.Edge(t, func(t *testing.T, ctx *testcontext.Context, planet *uitest.EdgePlanet, browser *rod.Browser) {
		t.Log("working")
	})
}
