// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package rangedlooptest

import (
	"context"

	"storx/satellite/metabase/rangedloop"
	"storx/satellite/metabase/segmentloop"
)

var _ rangedloop.RangeSplitter = (*InfiniteSegmentProvider)(nil)
var _ rangedloop.SegmentProvider = (*InfiniteSegmentProvider)(nil)

// InfiniteSegmentProvider allow to iterate indefinitely to test service cancellation.
type InfiniteSegmentProvider struct {
}

// CreateRanges splits the segments into equal ranges.
func (m *InfiniteSegmentProvider) CreateRanges(nRanges int, batchSize int) (segmentsProviders []rangedloop.SegmentProvider, err error) {
	for i := 0; i < nRanges; i++ {
		segmentsProviders = append(segmentsProviders, &InfiniteSegmentProvider{})
	}
	return segmentsProviders, nil
}

// Range returns range which is processed by this provider.
func (m *InfiniteSegmentProvider) Range() rangedloop.UUIDRange {
	return rangedloop.UUIDRange{}
}

// Iterate allows to loop over the segments stored in the provider.
func (m *InfiniteSegmentProvider) Iterate(ctx context.Context, fn func([]segmentloop.Segment) error) error {
	for {
		err := fn(make([]segmentloop.Segment, 3))
		if err != nil {
			return err
		}
	}
}
