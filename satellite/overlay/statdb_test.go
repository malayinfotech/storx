// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package overlay_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"common/pb"
	"common/storx"
	"common/storx/location"
	"common/testcontext"
	"storx/satellite"
	"storx/satellite/overlay"
	"storx/satellite/satellitedb/satellitedbtest"
)

func TestStatDB(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		testDatabase(ctx, t, db.OverlayCache())
	})
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		testDatabase(ctx, t, db.OverlayCache())
	})
}

func testDatabase(ctx context.Context, t *testing.T, cache overlay.DB) {
	{ // TestKnownUnreliableOrOffline and TestReliable
		for i, tt := range []struct {
			nodeID                storx.NodeID
			unknownAuditSuspended bool
			offlineSuspended      bool
			disqualified          bool
			offline               bool
			gracefullyexited      bool
			countryCode           string
		}{
			{storx.NodeID{1}, false, false, false, false, false, "DE"}, // good
			{storx.NodeID{2}, false, false, true, false, false, "DE"},  // disqualified
			{storx.NodeID{3}, true, false, false, false, false, "DE"},  // unknown audit suspended
			{storx.NodeID{4}, false, false, false, true, false, "DE"},  // offline
			{storx.NodeID{5}, false, false, false, false, true, "DE"},  // gracefully exited
			{storx.NodeID{6}, false, true, false, false, false, "DE"},  // offline suspended
			{storx.NodeID{7}, false, false, false, false, false, "FR"}, // excluded country
			{storx.NodeID{8}, false, false, false, false, false, ""},   // good
		} {
			addr := fmt.Sprintf("127.0.%d.0:8080", i)
			lastNet := fmt.Sprintf("127.0.%d", i)
			d := overlay.NodeCheckInInfo{
				NodeID:      tt.nodeID,
				Address:     &pb.NodeAddress{Address: addr},
				LastIPPort:  addr,
				LastNet:     lastNet,
				Version:     &pb.NodeVersion{Version: "v1.0.0"},
				Capacity:    &pb.NodeCapacity{},
				IsUp:        true,
				CountryCode: location.ToCountryCode(tt.countryCode),
			}
			err := cache.UpdateCheckIn(ctx, d, time.Now().UTC(), overlay.NodeSelectionConfig{})
			require.NoError(t, err)

			if tt.unknownAuditSuspended {
				err = cache.TestSuspendNodeUnknownAudit(ctx, tt.nodeID, time.Now())
				require.NoError(t, err)
			}

			if tt.offlineSuspended {
				err = cache.TestSuspendNodeOffline(ctx, tt.nodeID, time.Now())
				require.NoError(t, err)
			}

			if tt.disqualified {
				_, err = cache.DisqualifyNode(ctx, tt.nodeID, time.Now(), overlay.DisqualificationReasonUnknown)
				require.NoError(t, err)
			}
			if tt.offline {
				checkInInfo := getNodeInfo(tt.nodeID)
				err = cache.UpdateCheckIn(ctx, checkInInfo, time.Now().Add(-2*time.Hour), overlay.NodeSelectionConfig{})
				require.NoError(t, err)
			}
			if tt.gracefullyexited {
				req := &overlay.ExitStatusRequest{
					NodeID:              tt.nodeID,
					ExitInitiatedAt:     time.Now(),
					ExitLoopCompletedAt: time.Now(),
					ExitFinishedAt:      time.Now(),
				}
				_, err := cache.UpdateExitStatus(ctx, req)
				require.NoError(t, err)
			}
		}

		nodeIds := storx.NodeIDList{
			storx.NodeID{1}, storx.NodeID{2},
			storx.NodeID{3}, storx.NodeID{4},
			storx.NodeID{5}, storx.NodeID{6},
			storx.NodeID{7}, storx.NodeID{8},
			storx.NodeID{9},
		}
		criteria := &overlay.NodeCriteria{
			OnlineWindow:      time.Hour,
			ExcludedCountries: []string{"FR", "BE"},
		}

		invalid, err := cache.KnownUnreliableOrOffline(ctx, criteria, nodeIds)
		require.NoError(t, err)

		require.Contains(t, invalid, storx.NodeID{2}) // disqualified
		require.Contains(t, invalid, storx.NodeID{3}) // unknown audit suspended
		require.Contains(t, invalid, storx.NodeID{4}) // offline
		require.Contains(t, invalid, storx.NodeID{5}) // gracefully exited
		require.Contains(t, invalid, storx.NodeID{6}) // offline suspended
		require.Contains(t, invalid, storx.NodeID{9}) // not in db
		require.Len(t, invalid, 6)

		valid, err := cache.Reliable(ctx, criteria)
		require.NoError(t, err)

		require.NotContains(t, valid, storx.NodeID{2}) // disqualified
		require.NotContains(t, valid, storx.NodeID{3}) // unknown audit suspended
		require.NotContains(t, valid, storx.NodeID{4}) // offline
		require.NotContains(t, valid, storx.NodeID{5}) // gracefully exited
		require.NotContains(t, valid, storx.NodeID{6}) // offline suspended
		require.NotContains(t, valid, storx.NodeID{7}) // excluded country
		require.NotContains(t, valid, storx.NodeID{9}) // not in db
		require.Len(t, valid, 2)
	}

	{ // TestUpdateOperator
		nodeID := storx.NodeID{10}
		addr := "127.0.1.0:8080"
		lastNet := "127.0.1"
		d := overlay.NodeCheckInInfo{
			NodeID:     nodeID,
			Address:    &pb.NodeAddress{Address: addr},
			LastIPPort: addr,
			LastNet:    lastNet,
			Version:    &pb.NodeVersion{Version: "v1.0.0"},
			Capacity:   &pb.NodeCapacity{},
		}
		err := cache.UpdateCheckIn(ctx, d, time.Now().UTC(), overlay.NodeSelectionConfig{})
		require.NoError(t, err)

		update, err := cache.UpdateNodeInfo(ctx, nodeID, &overlay.InfoResponse{
			Operator: &pb.NodeOperator{
				Wallet:         "0x1111111111111111111111111111111111111111",
				Email:          "abc123@mail.test",
				WalletFeatures: []string{"wallet_features"},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, update)
		require.Equal(t, "0x1111111111111111111111111111111111111111", update.Operator.Wallet)
		require.Equal(t, "abc123@mail.test", update.Operator.Email)
		require.Equal(t, []string{"wallet_features"}, update.Operator.WalletFeatures)

		found, err := cache.Get(ctx, nodeID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, "0x1111111111111111111111111111111111111111", found.Operator.Wallet)
		require.Equal(t, "abc123@mail.test", found.Operator.Email)
		require.Equal(t, []string{"wallet_features"}, found.Operator.WalletFeatures)

		updateEmail, err := cache.UpdateNodeInfo(ctx, nodeID, &overlay.InfoResponse{
			Operator: &pb.NodeOperator{
				Wallet:         update.Operator.Wallet,
				Email:          "def456@mail.test",
				WalletFeatures: update.Operator.WalletFeatures,
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, updateEmail)
		assert.Equal(t, "0x1111111111111111111111111111111111111111", updateEmail.Operator.Wallet)
		assert.Equal(t, "def456@mail.test", updateEmail.Operator.Email)
		assert.Equal(t, []string{"wallet_features"}, updateEmail.Operator.WalletFeatures)

		updateWallet, err := cache.UpdateNodeInfo(ctx, nodeID, &overlay.InfoResponse{
			Operator: &pb.NodeOperator{
				Wallet:         "0x2222222222222222222222222222222222222222",
				Email:          updateEmail.Operator.Email,
				WalletFeatures: update.Operator.WalletFeatures,
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, updateWallet)
		assert.Equal(t, "0x2222222222222222222222222222222222222222", updateWallet.Operator.Wallet)
		assert.Equal(t, "def456@mail.test", updateWallet.Operator.Email)
		assert.Equal(t, []string{"wallet_features"}, updateWallet.Operator.WalletFeatures)

		updateWalletFeatures, err := cache.UpdateNodeInfo(ctx, nodeID, &overlay.InfoResponse{
			Operator: &pb.NodeOperator{
				Wallet:         updateWallet.Operator.Wallet,
				Email:          updateEmail.Operator.Email,
				WalletFeatures: []string{"wallet_features_updated"},
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, updateWalletFeatures)
		assert.Equal(t, "0x2222222222222222222222222222222222222222", updateWalletFeatures.Operator.Wallet)
		assert.Equal(t, "def456@mail.test", updateWalletFeatures.Operator.Email)
		assert.Equal(t, []string{"wallet_features_updated"}, updateWalletFeatures.Operator.WalletFeatures)
	}

	{ // test UpdateCheckIn updates the reputation correctly when the node is offline/online
		nodeID := storx.NodeID{1}

		// get the existing node info that is stored in nodes table
		_, err := cache.Get(ctx, nodeID)
		require.NoError(t, err)

		info := overlay.NodeCheckInInfo{
			NodeID: nodeID,
			Address: &pb.NodeAddress{
				Address: "1.2.3.4",
			},
			IsUp: false,
			Version: &pb.NodeVersion{
				Version:    "v0.0.0",
				CommitHash: "",
				Timestamp:  time.Time{},
				Release:    false,
			},
		}
		// update check-in when node is offline
		err = cache.UpdateCheckIn(ctx, info, time.Now(), overlay.NodeSelectionConfig{})
		require.NoError(t, err)
		_, err = cache.Get(ctx, nodeID)
		require.NoError(t, err)

		info.IsUp = true
		// update check-in when node is online
		err = cache.UpdateCheckIn(ctx, info, time.Now(), overlay.NodeSelectionConfig{})
		require.NoError(t, err)
		_, err = cache.Get(ctx, nodeID)
		require.NoError(t, err)

	}
}
