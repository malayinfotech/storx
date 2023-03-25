// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package audit

import (
	"context"
	"math/rand"

	"storx/satellite/metabase"
	"storx/satellite/metabase/segmentloop"
)

var _ segmentloop.Observer = (*Collector)(nil)

// Collector uses the segment loop to add segments to node reservoirs.
type Collector struct {
	Reservoirs map[metabase.NodeAlias]*Reservoir
	slotCount  int
	rand       *rand.Rand
}

// NewCollector instantiates a segment collector.
func NewCollector(reservoirSlots int, r *rand.Rand) *Collector {
	return &Collector{
		Reservoirs: make(map[metabase.NodeAlias]*Reservoir),
		slotCount:  reservoirSlots,
		rand:       r,
	}
}

// LoopStarted is called at each start of a loop.
func (collector *Collector) LoopStarted(context.Context, segmentloop.LoopInfo) (err error) {
	return nil
}

// RemoteSegment takes a remote segment found in metainfo and creates a reservoir for it if it doesn't exist already.
func (collector *Collector) RemoteSegment(ctx context.Context, segment *segmentloop.Segment) error {
	// we are expliticy not adding monitoring here as we are tracking loop observers separately

	for _, piece := range segment.AliasPieces {
		res, ok := collector.Reservoirs[piece.Alias]
		if !ok {
			res = NewReservoir(collector.slotCount)
			collector.Reservoirs[piece.Alias] = res
		}
		res.Sample(collector.rand, segment)
	}
	return nil
}

// InlineSegment returns nil because we're only auditing for storage nodes for now.
func (collector *Collector) InlineSegment(ctx context.Context, segment *segmentloop.Segment) (err error) {
	return nil
}

// Process performs per-node reservoir sampling on remote segments for addition into the audit queue.
func (collector *Collector) Process(ctx context.Context, segments []segmentloop.Segment) (err error) {
	for _, segment := range segments {
		// The reservoir ends up deferencing and copying the segment internally
		// but that's not obvious, so alias the loop variable.
		segment := segment
		if segment.Inline() {
			continue
		}
		if err := collector.RemoteSegment(ctx, &segment); err != nil {
			return err
		}
	}
	return nil
}
