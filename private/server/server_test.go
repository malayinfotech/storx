// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

package server_test

import (
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zeebo/errs"
	"go.uber.org/zap/zaptest"

	"common/errs2"
	"common/identity/testidentity"
	"common/peertls/tlsopts"
	"common/rpc"
	_ "common/rpc/quic"
	"common/storx"
	"common/sync2"
	"common/testcontext"
	"storx/private/server"
	"storx/private/testplanet"
)

func TestServer(t *testing.T) {
	ctx := testcontext.New(t)
	log := zaptest.NewLogger(t)
	identity := testidentity.MustPregeneratedIdentity(0, storx.LatestIDVersion())

	host := "127.0.0.1"
	if hostlist := os.Getenv("STORX_TEST_HOST"); hostlist != "" {
		host, _, _ = strings.Cut(hostlist, ";")
	}

	tlsOptions, err := tlsopts.NewOptions(identity, tlsopts.Config{
		PeerIDVersions: "latest",
	}, nil)
	require.NoError(t, err)

	instance, err := server.New(log.Named("server"), tlsOptions, server.Config{
		Address:          host + ":0",
		PrivateAddress:   host + ":0",
		TCPFastOpen:      true,
		TCPFastOpenQueue: 256,
	})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, instance.Close())
	}()

	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	require.Empty(t, sync2.Concurrently(
		func() error {
			err = instance.Run(serverCtx)
			return errs2.IgnoreCanceled(err)
		},
		func() (err error) {
			defer serverCancel()

			dialer := net.Dialer{}
			conn, err := dialer.DialContext(ctx, "tcp", instance.PrivateAddr().String())
			if err != nil {
				return errs.Wrap(err)
			}
			defer func() { err = errs.Combine(err, conn.Close()) }()

			_, err = conn.Write([]byte{1})
			return errs.Wrap(err)
		},
	))
}

func TestHybridConnector_Basic(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount:   1,
		StorageNodeCount: 0,
		UplinkCount:      1,
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		sat := planet.Satellites[0]
		dialer := planet.Uplinks[0].Dialer

		dialer.Connector = rpc.NewHybridConnector()

		conn, err := dialer.Connector.DialContext(ctx, dialer.TLSOptions.ClientTLSConfig(sat.ID()), sat.Addr())
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	})
}

func TestHybridConnector_QUICOnly(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount:   1,
		StorageNodeCount: 0,
		UplinkCount:      0,
		Reconfigure:      testplanet.DisableTCP,
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		sat := planet.Satellites[0]
		identity, err := planet.NewIdentity()
		require.NoError(t, err)

		tlsOptions, err := tlsopts.NewOptions(identity, tlsopts.Config{
			PeerIDVersions: strconv.Itoa(int(storx.LatestIDVersion().Number)),
		}, nil)
		require.NoError(t, err)

		connector := rpc.NewHybridConnector()

		conn, err := connector.DialContext(ctx, tlsOptions.ClientTLSConfig(sat.ID()), sat.Addr())
		require.NoError(t, err)
		require.Equal(t, "udp", conn.LocalAddr().Network())
		require.NoError(t, conn.Close())
	})
}

func TestHybridConnector_TCPOnly(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount:   1,
		StorageNodeCount: 0,
		UplinkCount:      0,
		Reconfigure:      testplanet.DisableQUIC,
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		sat := planet.Satellites[0]
		identity, err := planet.NewIdentity()
		require.NoError(t, err)

		tlsOptions, err := tlsopts.NewOptions(identity, tlsopts.Config{
			PeerIDVersions: strconv.Itoa(int(storx.LatestIDVersion().Number)),
		}, nil)
		require.NoError(t, err)

		connector := rpc.NewHybridConnector()

		conn, err := connector.DialContext(ctx, tlsOptions.ClientTLSConfig(sat.ID()), sat.Addr())
		require.NoError(t, err)
		require.Equal(t, "tcp", conn.LocalAddr().Network())
		require.NoError(t, conn.Close())
	})
}
