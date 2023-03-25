// Copyright (C) 2019 storx Labs, Inc.
// See LICENSE for copying information.

package reports_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"common/uuid"
	"storx/cmd/satellite/reports"
	"storx/satellite/attribution"
)

func TestProcessAttributions(t *testing.T) {
	log := zaptest.NewLogger(t)

	requireSum := func(total reports.Total, n int) {
		require.Equal(t, float64(n), total.ByteHours)
		require.Equal(t, float64(n), total.SegmentHours)
		require.Equal(t, float64(n), total.ObjectHours)
		require.Equal(t, float64(n), total.BucketHours)
		require.Equal(t, int64(n), total.BytesEgress)
	}

	newUsage := func(userAgent string, projectID uuid.UUID, bucketName string) *attribution.BucketUsage {
		return &attribution.BucketUsage{
			UserAgent:    []byte(userAgent),
			ProjectID:    projectID.Bytes(),
			BucketName:   []byte(bucketName),
			ByteHours:    1,
			SegmentHours: 1,
			ObjectHours:  1,
			Hours:        1,
			EgressData:   1,
		}
	}

	id, err := uuid.New()
	require.NoError(t, err)

	// test empty user agents
	attributions := []*attribution.BucketUsage{
		newUsage("", id, ""),
		{
			ByteHours:    1,
			SegmentHours: 1,
			ObjectHours:  1,
			Hours:        1,
			EgressData:   1,
		},
	}
	totals := reports.ProcessAttributions(attributions, nil, log)
	require.Equal(t, 0, len(totals))

	// test user agent with additional entries and uppercase letters is summed with
	// the first one
	attributions = []*attribution.BucketUsage{
		newUsage("teststorx", id, ""),
		newUsage("TESTSTORX/other", id, ""),
	}
	totals = reports.ProcessAttributions(attributions, nil, log)
	require.Equal(t, 1, len(totals))
	requireSum(totals[reports.AttributionTotalsIndex{"teststorx", id.String(), ""}], 2)

	// test two user agents are summed separately
	attributions = []*attribution.BucketUsage{
		newUsage("teststorx1", id, ""),
		newUsage("teststorx1", id, ""),
		newUsage("teststorx2", id, ""),
		newUsage("teststorx2", id, ""),
	}
	totals = reports.ProcessAttributions(attributions, nil, log)
	require.Equal(t, 2, len(totals))
	requireSum(totals[reports.AttributionTotalsIndex{"teststorx1", id.String(), ""}], 2)
	requireSum(totals[reports.AttributionTotalsIndex{"teststorx2", id.String(), ""}], 2)

	// Test that different project IDs are summed separately
	id2, err := uuid.New()
	require.NoError(t, err)
	attributions = []*attribution.BucketUsage{
		newUsage("teststorx1", id, ""),
		newUsage("teststorx1", id, ""),
		newUsage("teststorx1", id2, ""),
	}
	totals = reports.ProcessAttributions(attributions, nil, log)
	require.Equal(t, 2, len(totals))
	requireSum(totals[reports.AttributionTotalsIndex{"teststorx1", id.String(), ""}], 2)
	requireSum(totals[reports.AttributionTotalsIndex{"teststorx1", id2.String(), ""}], 1)

	// Test that different bucket names are summed separately
	attributions = []*attribution.BucketUsage{
		newUsage("teststorx1", id, "1"),
		newUsage("teststorx1", id, "1"),
		newUsage("teststorx1", id, "2"),
	}
	totals = reports.ProcessAttributions(attributions, nil, log)
	require.Equal(t, 2, len(totals))
	requireSum(totals[reports.AttributionTotalsIndex{"teststorx1", id.String(), "1"}], 2)
	requireSum(totals[reports.AttributionTotalsIndex{"teststorx1", id.String(), "2"}], 1)

	// Test that unspecified user agents are filtered out
	attributions = []*attribution.BucketUsage{
		newUsage("teststorx1", id, ""),
		newUsage("teststorx2", id, ""),
		newUsage("teststorx3", id, ""),
	}
	totals = reports.ProcessAttributions(attributions, []string{"teststorx1", "teststorx3"}, log)
	require.Equal(t, 2, len(totals))
	require.Contains(t, totals, reports.AttributionTotalsIndex{"teststorx1", id.String(), ""})
	require.Contains(t, totals, reports.AttributionTotalsIndex{"teststorx3", id.String(), ""})
}
