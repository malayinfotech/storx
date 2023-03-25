// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package satellite

import (
	"context"
	"errors"
	"net"
	"runtime/pprof"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/zeebo/errs"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"common/identity"
	"common/storx"
	"private/debug"
	"private/version"
	"storx/private/lifecycle"
	"storx/private/version/checker"
	"storx/satellite/admin"
	"storx/satellite/analytics"
	"storx/satellite/buckets"
	"storx/satellite/console"
	"storx/satellite/console/restkeys"
	"storx/satellite/metabase"
	"storx/satellite/payments"
	"storx/satellite/payments/stripecoinpayments"
)

// Admin is the satellite core process that runs chores.
//
// architecture: Peer
type Admin struct {
	// core dependencies
	Log        *zap.Logger
	Identity   *identity.FullIdentity
	DB         DB
	MetabaseDB *metabase.DB

	Servers  *lifecycle.Group
	Services *lifecycle.Group

	Debug struct {
		Listener net.Listener
		Server   *debug.Server
	}

	Version struct {
		Chore   *checker.Chore
		Service *checker.Service
	}

	Payments struct {
		Accounts payments.Accounts
		Service  *stripecoinpayments.Service
		Stripe   stripecoinpayments.StripeClient
	}

	Admin struct {
		Listener net.Listener
		Server   *admin.Server
	}

	Buckets struct {
		Service *buckets.Service
	}

	REST struct {
		Keys *restkeys.Service
	}

	FreezeAccounts struct {
		Service *console.AccountFreezeService
	}
}

// NewAdmin creates a new satellite admin peer.
func NewAdmin(log *zap.Logger, full *identity.FullIdentity, db DB, metabaseDB *metabase.DB,
	versionInfo version.Info, config *Config, atomicLogLevel *zap.AtomicLevel) (*Admin, error) {
	peer := &Admin{
		Log:        log,
		Identity:   full,
		DB:         db,
		MetabaseDB: metabaseDB,

		Servers:  lifecycle.NewGroup(log.Named("servers")),
		Services: lifecycle.NewGroup(log.Named("services")),
	}

	{
		peer.Buckets.Service = buckets.NewService(db.Buckets(), metabaseDB)
	}

	{ // setup rest keys
		peer.REST.Keys = restkeys.NewService(db.OIDC().OAuthTokens(), config.RESTKeys)
	}

	{ // setup debug
		var err error
		if config.Debug.Address != "" {
			peer.Debug.Listener, err = net.Listen("tcp", config.Debug.Address)
			if err != nil {
				withoutStack := errors.New(err.Error())
				peer.Log.Debug("failed to start debug endpoints", zap.Error(withoutStack))
			}
		}
		debugConfig := config.Debug
		debugConfig.ControlTitle = "Admin"
		peer.Debug.Server = debug.NewServerWithAtomicLevel(log.Named("debug"), peer.Debug.Listener, monkit.Default, debugConfig, atomicLogLevel)
		peer.Servers.Add(lifecycle.Item{
			Name:  "debug",
			Run:   peer.Debug.Server.Run,
			Close: peer.Debug.Server.Close,
		})
	}

	{
		if !versionInfo.IsZero() {
			peer.Log.Debug("Version info",
				zap.Stringer("Version", versionInfo.Version.Version),
				zap.String("Commit Hash", versionInfo.CommitHash),
				zap.Stringer("Build Timestamp", versionInfo.Timestamp),
				zap.Bool("Release Build", versionInfo.Release),
			)
		}
		peer.Version.Service = checker.NewService(log.Named("version"), config.Version, versionInfo, "Satellite")
		peer.Version.Chore = checker.NewChore(peer.Version.Service, config.Version.CheckInterval)

		peer.Services.Add(lifecycle.Item{
			Name: "version",
			Run:  peer.Version.Chore.Run,
		})
	}

	{ // setup payments
		pc := config.Payments

		var stripeClient stripecoinpayments.StripeClient
		switch pc.Provider {
		case "": // just new mock, only used in testing binaries
			stripeClient = stripecoinpayments.NewStripeMock(
				peer.DB.StripeCoinPayments().Customers(),
				peer.DB.Console().Users(),
			)
		case "mock":
			stripeClient = pc.MockProvider
		case "stripecoinpayments":
			stripeClient = stripecoinpayments.NewStripeClient(log, pc.StripeCoinPayments)
		default:
			return nil, errs.New("invalid stripe coin payments provider %q", pc.Provider)
		}

		prices, err := pc.UsagePrice.ToModel()
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		priceOverrides, err := pc.UsagePriceOverrides.ToModels()
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		peer.Payments.Service, err = stripecoinpayments.NewService(
			peer.Log.Named("payments.stripe:service"),
			stripeClient,
			pc.StripeCoinPayments,
			peer.DB.StripeCoinPayments(),
			peer.DB.Wallets(),
			peer.DB.Billing(),
			peer.DB.Console().Projects(),
			peer.DB.Console().Users(),
			peer.DB.ProjectAccounting(),
			prices,
			priceOverrides,
			pc.PackagePlans.Packages,
			pc.BonusRate)

		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		peer.Payments.Stripe = stripeClient
		peer.Payments.Accounts = peer.Payments.Service.Accounts()
		peer.FreezeAccounts.Service = console.NewAccountFreezeService(
			db.Console().AccountFreezeEvents(),
			db.Console().Users(),
			db.Console().Projects(),
			analytics.NewService(peer.Log.Named("analytics:service"), config.Analytics, config.Console.SatelliteName),
		)
	}

	{ // setup admin endpoint
		var err error
		peer.Admin.Listener, err = net.Listen("tcp", config.Admin.Address)
		if err != nil {
			return nil, err
		}

		adminConfig := config.Admin
		adminConfig.AuthorizationToken = config.Console.AuthToken

		peer.Admin.Server = admin.NewServer(log.Named("admin"), peer.Admin.Listener, peer.DB, peer.Buckets.Service, peer.REST.Keys, peer.FreezeAccounts.Service, peer.Payments.Accounts, config.Console, adminConfig)
		peer.Servers.Add(lifecycle.Item{
			Name:  "admin",
			Run:   peer.Admin.Server.Run,
			Close: peer.Admin.Server.Close,
		})
	}

	return peer, nil
}

// Run runs satellite until it's either closed or it errors.
func (peer *Admin) Run(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)

	group, ctx := errgroup.WithContext(ctx)

	pprof.Do(ctx, pprof.Labels("subsystem", "admin"), func(ctx context.Context) {
		peer.Servers.Run(ctx, group)
		peer.Services.Run(ctx, group)

		pprof.Do(ctx, pprof.Labels("name", "subsystem-wait"), func(ctx context.Context) {
			err = group.Wait()
		})
	})
	return err
}

// Close closes all the resources.
func (peer *Admin) Close() error {
	return errs.Combine(
		peer.Servers.Close(),
		peer.Services.Close(),
	)
}

// ID returns the peer ID.
func (peer *Admin) ID() storx.NodeID { return peer.Identity.ID }
