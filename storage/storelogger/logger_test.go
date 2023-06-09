// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package storelogger

import (
	"testing"

	"go.uber.org/zap"

	"storx/storage/teststore"
	"storx/storage/testsuite"
)

func TestSuite(t *testing.T) {
	store := teststore.New()
	store.SetLookupLimit(500)
	logged := New(zap.NewNop(), store)
	testsuite.RunTests(t, logged)
}

func BenchmarkSuite(b *testing.B) {
	store := teststore.New()
	logged := New(zap.NewNop(), store)
	testsuite.RunBenchmarks(b, logged)
}
