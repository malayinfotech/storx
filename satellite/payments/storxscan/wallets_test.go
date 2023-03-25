// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package storxscan_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"common/testcontext"
	"common/testrand"
	"storx/private/blockchain"
	"storx/satellite"
	"storx/satellite/satellitedb/satellitedbtest"
)

func TestWalletsDB(t *testing.T) {
	userID1 := testrand.UUID()
	userID2 := testrand.UUID()
	userID3 := testrand.UUID()
	walletAddress1, err := blockchain.BytesToAddress(testrand.Bytes(20))
	require.NoError(t, err)
	walletAddress2, err := blockchain.BytesToAddress(testrand.Bytes(20))
	require.NoError(t, err)
	walletAddress3, err := blockchain.BytesToAddress(testrand.Bytes(20))
	require.NoError(t, err)

	t.Run("get wallet", func(t *testing.T) {
		satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
			err := db.Wallets().Add(ctx, userID1, walletAddress1)
			require.NoError(t, err)
			err = db.Wallets().Add(ctx, userID2, walletAddress2)
			require.NoError(t, err)
			err = db.Wallets().Add(ctx, userID3, walletAddress3)
			require.NoError(t, err)

			address1, err := db.Wallets().GetWallet(ctx, userID1)
			require.NoError(t, err)
			require.Equal(t, walletAddress1, address1)
			address2, err := db.Wallets().GetWallet(ctx, userID2)
			require.NoError(t, err)
			require.Equal(t, walletAddress2, address2)
			address3, err := db.Wallets().GetWallet(ctx, userID3)
			require.NoError(t, err)
			require.Equal(t, walletAddress3, address3)
		})
	})
}
