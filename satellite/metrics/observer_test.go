// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"common/testcontext"
	"common/uuid"
	"storx/satellite/metabase"
	"storx/satellite/metabase/rangedloop"
	"storx/satellite/metabase/rangedloop/rangedlooptest"
	"storx/satellite/metabase/segmentloop"
)

var (
	inline1 = []segmentloop.Segment{
		{StreamID: uuid.UUID{1}, EncryptedSize: 10},
	}
	remote2 = []segmentloop.Segment{
		{StreamID: uuid.UUID{2}, EncryptedSize: 16, Pieces: metabase.Pieces{{}}},
		{StreamID: uuid.UUID{2}, EncryptedSize: 10},
	}
	remote3 = []segmentloop.Segment{
		{StreamID: uuid.UUID{3}, EncryptedSize: 16, Pieces: metabase.Pieces{{}}},
		{StreamID: uuid.UUID{3}, EncryptedSize: 16, Pieces: metabase.Pieces{{}}},
		{StreamID: uuid.UUID{3}, EncryptedSize: 16, Pieces: metabase.Pieces{{}}},
		{StreamID: uuid.UUID{3}, EncryptedSize: 10},
	}
)

func TestObserver(t *testing.T) {
	ctx := testcontext.New(t)

	loop := func(tb testing.TB, obs *Observer, streams ...[]segmentloop.Segment) Metrics {
		service := rangedloop.NewService(
			zap.NewNop(),
			rangedloop.Config{BatchSize: 2, Parallelism: 2},
			&rangedlooptest.RangeSplitter{Segments: combineSegments(streams...)},
			[]rangedloop.Observer{obs})
		_, err := service.RunOnce(ctx)
		require.NoError(tb, err)
		return obs.TestingMetrics()
	}

	t.Run("stats aggregation", func(t *testing.T) {
		obs := NewObserver()

		metrics := loop(t, obs, inline1, remote2, remote3)

		require.Equal(t, Metrics{
			InlineObjects:       1,
			RemoteObjects:       2,
			TotalInlineSegments: 3,
			TotalRemoteSegments: 4,
			TotalInlineBytes:    30,
			TotalRemoteBytes:    64,
		}, metrics)
	})

	t.Run("stats reset by start", func(t *testing.T) {
		obs := NewObserver()

		_ = loop(t, obs, inline1)

		// Any metrics gathered during the first loop should be dropped.
		metrics := loop(t, obs, remote3)

		require.Equal(t, Metrics{
			InlineObjects:       0,
			RemoteObjects:       1,
			TotalInlineSegments: 1,
			TotalRemoteSegments: 3,
			TotalInlineBytes:    10,
			TotalRemoteBytes:    48,
		}, metrics)
	})

	t.Run("join fails gracefully on bad partial type", func(t *testing.T) {
		type wrongPartial struct{ rangedloop.Partial }
		obs := NewObserver()
		err := obs.Start(ctx, time.Time{})
		require.NoError(t, err)
		err = obs.Join(ctx, wrongPartial{})
		require.EqualError(t, err, "metrics: expected *metrics.observerFork but got metrics.wrongPartial")
	})
}

func combineSegments(ss ...[]segmentloop.Segment) []segmentloop.Segment {
	var combined []segmentloop.Segment
	for _, s := range ss {
		combined = append(combined, s...)
	}
	return combined
}
