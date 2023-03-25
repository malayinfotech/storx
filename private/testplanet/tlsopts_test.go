// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package testplanet_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"common/identity"
	"common/identity/testidentity"
	"common/peertls/tlsopts"
	"common/rpc"
	"common/storx"
	"common/testcontext"
	"storx/private/testplanet"
)

func TestOptions_ServerOption_Peer_CA_Whitelist(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 0, StorageNodeCount: 2, UplinkCount: 0,
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		sn := planet.StorageNodes[1]
		testidentity.CompleteIdentityVersionsTest(t, func(t *testing.T, version storx.IDVersion, ident *identity.FullIdentity) {
			tlsOptions, err := tlsopts.NewOptions(ident, tlsopts.Config{
				PeerIDVersions: "*",
			}, nil)
			require.NoError(t, err)

			dialer := rpc.NewDefaultDialer(tlsOptions)

			conn, err := dialer.DialNodeURL(ctx, sn.NodeURL())
			assert.NotNil(t, conn)
			assert.NoError(t, err)

			assert.NoError(t, conn.Close())
		})
	})
}
