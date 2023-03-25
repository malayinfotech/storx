// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package teststore

import (
	"testing"

	"storx/storage/testsuite"
)

func TestSuite(t *testing.T) {
	store := New()
	store.SetLookupLimit(500)
	testsuite.RunTests(t, store)
}
func BenchmarkSuite(b *testing.B) {
	testsuite.RunBenchmarks(b, New())
}
