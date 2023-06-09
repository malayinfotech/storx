// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package piecedeletion_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"common/storx"
	"common/sync2"
	"common/testcontext"
	"storx/satellite/metainfo/piecedeletion"
)

type HandleLimitVerifier struct {
	Active        int64
	ExpectedLimit int64
}

func (*HandleLimitVerifier) NewQueue() piecedeletion.Queue {
	panic("should not be called")
}

func (verifier *HandleLimitVerifier) Handle(ctx context.Context, node storx.NodeURL, queue piecedeletion.Queue) {
	current := atomic.AddInt64(&verifier.Active, 1)
	if current > verifier.ExpectedLimit {
		panic("over limit")
	}
	defer atomic.AddInt64(&verifier.Active, -1)
	defer sync2.Sleep(ctx, time.Millisecond)
}

func TestLimitedHandler(t *testing.T) {
	ctx := testcontext.New(t)
	defer ctx.Cleanup()

	verifier := &HandleLimitVerifier{
		Active:        0,
		ExpectedLimit: 8,
	}

	limited := piecedeletion.NewLimitedHandler(verifier, int(verifier.ExpectedLimit))

	for i := 0; i < 800; i++ {
		ctx.Go(func() error {
			limited.Handle(ctx, storx.NodeURL{}, nil)
			return nil
		})
	}
}
