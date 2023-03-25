// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package storxscantest

import (
	"context"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/zeebo/errs"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"common/grant"
	"common/storx"
	"common/testcontext"
	"private/dbutil/pgtest"
	"storx/private/blockchain"
	"storx/private/testmonkit"
	"storx/private/testplanet"
	"storx/satellite"
	"storx/satellite/satellitedb/satellitedbtest"
	"storxscan"
	"storxscan/private/testeth"
	"storxscan/storxscandb/storxscandbtest"
	"uplink"
)

// Stack contains references to storxscan app and eth test network.
type Stack struct {
	Log      *zap.Logger
	App      *storxscan.App
	StartApp func() error
	CloseApp func() error
	Network  *testeth.Network
	Token    blockchain.Address
}

// Test defines common services for storxscan tests.
type Test func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet, stack *Stack)

// Run runs testplanet and storxscan and executes test function.
func Run(t *testing.T, test Test) {
	databases := satellitedbtest.Databases()
	if len(databases) == 0 {
		t.Fatal("Databases flag missing, set at least one:\n" +
			"-postgres-test-db=" + pgtest.DefaultPostgres + "\n" +
			"-cockroach-test-db=" + pgtest.DefaultCockroach)
	}

	config := testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 4, UplinkCount: 1,
		NonParallel: true,
	}

	for _, satelliteDB := range databases {
		satelliteDB := satelliteDB
		t.Run(satelliteDB.Name, func(t *testing.T) {
			parallel := !config.NonParallel
			if parallel {
				t.Parallel()
			}

			if satelliteDB.MasterDB.URL == "" {
				t.Skipf("Database %s connection string not provided. %s", satelliteDB.MasterDB.Name, satelliteDB.MasterDB.Message)
			}
			planetConfig := config
			if planetConfig.Name == "" {
				planetConfig.Name = t.Name()
			}

			log := testplanet.NewLogger(t)

			testmonkit.Run(context.Background(), t, func(parent context.Context) {
				defer pprof.SetGoroutineLabels(parent)
				parent = pprof.WithLabels(parent, pprof.Labels("test", t.Name()))

				timeout := config.Timeout
				if timeout == 0 {
					timeout = testcontext.DefaultTimeout
				}
				ctx := testcontext.NewWithContextAndTimeout(parent, t, timeout)
				defer ctx.Cleanup()

				// storxscan ---------
				stack := Stack{
					Log: log.Named("storxscan"),
				}

				storxscanDB, err := storxscandbtest.OpenDB(ctx, stack.Log.Named("db"), satelliteDB.MasterDB.URL, "storxscandb-"+t.Name(), "S")
				if err != nil {
					t.Fatalf("%+v", err)
				}
				defer ctx.Check(storxscanDB.Close)

				if err = storxscanDB.MigrateToLatest(ctx); err != nil {
					t.Fatalf("%+v", err)
				}

				stack.Network, err = testeth.NewNetwork()
				if err != nil {
					t.Fatalf("%+v", err)
				}
				if err = stack.Network.Start(); err != nil {
					t.Fatalf("%+v", err)
				}
				defer ctx.Check(stack.Network.Close)

				token, err := testeth.DeployToken(ctx, stack.Network, 1000000)
				if err != nil {
					t.Fatalf("%+v", err)
				}
				stack.Token = blockchain.Address(token)

				var storxscanConfig storxscan.Config
				storxscanConfig.API.Address = "127.0.0.1:0"
				storxscanConfig.API.Keys = []string{"eu:eusecret"}
				storxscanConfig.Tokens.Endpoint = stack.Network.HTTPEndpoint()
				storxscanConfig.Tokens.Contract = stack.Token.Hex()
				storxscanConfig.TokenPrice.PriceWindow = time.Minute
				storxscanConfig.TokenPrice.Interval = time.Minute
				storxscanConfig.TokenPrice.UseTestPrices = true

				stack.App, err = storxscan.NewApp(stack.Log.Named("app"), storxscanConfig, storxscanDB)
				if err != nil {
					t.Fatalf("%+v", err)
				}

				var run errgroup.Group
				runCtx, runCancel := context.WithCancel(ctx)

				stack.StartApp = func() error {
					storxscanConfig.API.Address = stack.App.API.Listener.Addr().String()

					stack.App, err = storxscan.NewApp(stack.Log.Named("app"), storxscanConfig, storxscanDB)
					if err != nil {
						return err
					}

					runCtx, runCancel = context.WithCancel(ctx)

					run = errgroup.Group{}
					run.Go(func() error {
						err := stack.App.Run(runCtx)
						return err
					})

					return nil
				}
				stack.CloseApp = func() error {
					runCancel()

					var errlist errs.Group
					errlist.Add(run.Wait())
					errlist.Add(stack.App.Close())
					return errlist.Err()
				}

				run.Go(func() error {
					err := stack.App.Run(runCtx)
					return err
				})
				defer ctx.Check(stack.CloseApp)
				// ------------

				planetConfig.Reconfigure = testplanet.Reconfigure{Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
					config.Payments.Storxscan.Auth.Identifier = "eu"
					config.Payments.Storxscan.Auth.Secret = "eusecret"
					config.Payments.Storxscan.Endpoint = "http://" + stack.App.API.Listener.Addr().String()
					config.Payments.Storxscan.Confirmations = 1
					config.Payments.Storxscan.DisableLoop = false
				}}

				planet, err := testplanet.NewCustom(ctx, log, planetConfig, satelliteDB)
				if err != nil {
					t.Fatalf("%+v", err)
				}
				defer ctx.Check(planet.Shutdown)

				planet.Start(ctx)
				provisionUplinks(ctx, t, planet)

				test(t, ctx, planet, &stack)
			})
		})
	}
}

func provisionUplinks(ctx context.Context, t *testing.T, planet *testplanet.Planet) {
	for _, planetUplink := range planet.Uplinks {
		for _, satellite := range planet.Satellites {
			apiKey := planetUplink.APIKey[satellite.ID()]

			// create access grant manually to avoid dialing satellite for
			// project id and deriving key with argon2.IDKey method
			encAccess := grant.NewEncryptionAccessWithDefaultKey(&storx.Key{})
			encAccess.SetDefaultPathCipher(storx.EncAESGCM)

			grantAccess := grant.Access{
				SatelliteAddress: satellite.URL(),
				APIKey:           apiKey,
				EncAccess:        encAccess,
			}

			serializedAccess, err := grantAccess.Serialize()
			if err != nil {
				t.Fatalf("%+v", err)
			}
			access, err := uplink.ParseAccess(serializedAccess)
			if err != nil {
				t.Fatalf("%+v", err)
			}

			planetUplink.Access[satellite.ID()] = access
		}
	}
}
