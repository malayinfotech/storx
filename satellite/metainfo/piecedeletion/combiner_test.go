// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package piecedeletion_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"common/storx"
	"common/sync2"
	"common/testcontext"
	"common/testrand"
	"storx/satellite/metainfo/piecedeletion"
)

type CountHandler struct {
	Count int64
}

func (handler *CountHandler) Handle(ctx context.Context, node storx.NodeURL, queue piecedeletion.Queue) {
	for {
		list, ok := queue.PopAll()
		if !ok {
			return
		}
		for _, job := range list {
			atomic.AddInt64(&handler.Count, int64(len(job.Pieces)))
			job.Resolve.Success()
		}
	}
}

func TestCombiner(t *testing.T) {
	ctx := testcontext.New(t)
	defer ctx.Cleanup()

	const (
		activeLimit   = 8
		nodeCount     = 70
		requestCount  = 100
		parallelCount = 10
		queueSize     = 5
	)

	nodes := []storx.NodeURL{}
	for i := 0; i < nodeCount; i++ {
		nodes = append(nodes, storx.NodeURL{
			ID: testrand.NodeID(),
		})
	}

	counter := &CountHandler{}
	limited := piecedeletion.NewLimitedHandler(counter, activeLimit)
	newQueue := func() piecedeletion.Queue {
		return piecedeletion.NewLimitedJobs(queueSize)
	}

	combiner := piecedeletion.NewCombiner(ctx, limited, newQueue)

	var wg sync.WaitGroup
	for i := 0; i < parallelCount; i++ {
		wg.Add(1)
		ctx.Go(func() error {
			defer wg.Done()

			pending, err := sync2.NewSuccessThreshold(requestCount, 0.999999)
			if err != nil {
				return err
			}

			for k := 0; k < requestCount; k++ {
				node := nodes[testrand.Intn(len(nodes))]

				combiner.Enqueue(node, piecedeletion.Job{
					Pieces:  []storx.PieceID{testrand.PieceID()},
					Resolve: pending,
				})
			}

			pending.Wait(ctx)
			return nil
		})
	}
	wg.Wait()
	combiner.Close()

	require.Equal(t, int(counter.Count), int(requestCount*parallelCount))
}
