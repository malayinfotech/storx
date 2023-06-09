// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package checker_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"common/pb"
	"common/storx"
	"storx/satellite/repair/checker"
)

func TestRepairOverrideConfigValidation(t *testing.T) {
	tests := []struct {
		description    string
		overrideConfig string
		expectError    bool
		size           int
	}{
		{
			description:    "valid multi repair override config",
			overrideConfig: "2/5/20-3,1/4/10-2",
			expectError:    false,
			size:           2,
		},
		{
			description:    "valid single repair override config",
			overrideConfig: "2/5/20-3",
			expectError:    false,
			size:           1,
		},
		{
			description:    "invalid repair override config - numbers decrease",
			overrideConfig: "1/5/4-3",
			expectError:    true,
		},
		{
			description:    "invalid repair override config - starts at 0",
			overrideConfig: "0/5/6-3",
			expectError:    true,
		},
		{
			description:    "invalid repair override config - strings",
			overrideConfig: "1/2/4-a",
			expectError:    true,
		},
		{
			description:    "invalid repair override config - strings",
			overrideConfig: "1/b/4-3",
			expectError:    true,
		},
		{
			description:    "invalid repair override config - floating point numbers",
			overrideConfig: "2/3.2/4-3",
			expectError:    true,
		},
		{
			description:    "invalid repair override config - floating point numbers",
			overrideConfig: "1/5/6-3.2",
			expectError:    true,
		},
		{
			description:    "invalid repair override config - no override value",
			overrideConfig: "1/2/4",
			expectError:    true,
		},
		{
			description:    "invalid repair override config - not enough rs numbers",
			overrideConfig: "1/6-3",
			expectError:    true,
		},
		{
			description:    "invalid repair override config - override < min",
			overrideConfig: "2/5/20-1",
			expectError:    true,
		},
		{
			description:    "invalid repair override config - override >= optimal",
			overrideConfig: "2/5/20-5",
			expectError:    true,
		},
		{
			description:    "valid repair override config - empty items in multi value",
			overrideConfig: ",2/5/20-4,,3/6/7-4",
			expectError:    false,
			size:           2,
		},
		{
			description:    "valid repair override config - empty",
			overrideConfig: "",
			expectError:    false,
			size:           0,
		},
	}

	for _, tt := range tests {
		t.Log(tt.description)

		newOverrides := checker.RepairOverrides{}
		err := newOverrides.Set(tt.overrideConfig)
		if tt.expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Len(t, newOverrides.List, tt.size)
		}

	}
}

func TestRepairOverride(t *testing.T) {
	overrideConfig := "29/80/95-52,10/30/40-25"
	newOverrides := checker.RepairOverrides{}
	err := newOverrides.Set(overrideConfig)
	require.NoError(t, err)

	schemes := [][]int16{
		{10, 20, 30, 40},
		{29, 35, 80, 95},
		{29, 60, 80, 95},
		{2, 5, 10, 30},
	}
	storxSchemes := []storx.RedundancyScheme{}
	pbSchemes := []*pb.RedundancyScheme{}
	for _, scheme := range schemes {
		newStorx := storx.RedundancyScheme{
			RequiredShares: scheme[0],
			RepairShares:   scheme[1],
			OptimalShares:  scheme[2],
			TotalShares:    scheme[3],
		}
		storxSchemes = append(storxSchemes, newStorx)

		newPB := &pb.RedundancyScheme{
			MinReq:           int32(scheme[0]),
			RepairThreshold:  int32(scheme[1]),
			SuccessThreshold: int32(scheme[2]),
			Total:            int32(scheme[3]),
		}
		pbSchemes = append(pbSchemes, newPB)
	}

	ro := newOverrides.GetMap()
	require.EqualValues(t, 25, ro.GetOverrideValue(storxSchemes[0]))
	require.EqualValues(t, 25, ro.GetOverrideValuePB(pbSchemes[0]))

	// second and third schemes should have the same override value (52) despite having a different repair threshold.
	require.EqualValues(t, 52, ro.GetOverrideValue(storxSchemes[1]))
	require.EqualValues(t, 52, ro.GetOverrideValuePB(pbSchemes[1]))
	require.EqualValues(t, 52, ro.GetOverrideValue(storxSchemes[2]))
	require.EqualValues(t, 52, ro.GetOverrideValuePB(pbSchemes[2]))

	// fourth scheme has no matching override config.
	require.EqualValues(t, 0, ro.GetOverrideValue(storxSchemes[3]))
	require.EqualValues(t, 0, ro.GetOverrideValuePB(pbSchemes[3]))
}
