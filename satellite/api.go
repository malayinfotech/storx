// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package satellite

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime/pprof"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/zeebo/errs"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"common/identity"
	"common/pb"
	"common/peertls/extensions"
	"common/peertls/tlsopts"
	"common/rpc"
	"common/signing"
	"common/storx"
	"private/debug"
	"private/version"
	"storx/private/lifecycle"
	"storx/private/server"
	"storx/private/version/checker"
	"storx/satellite/abtesting"
	"storx/satellite/accounting"
	"storx/satellite/analytics"
	"storx/satellite/buckets"
	"storx/satellite/console"
	"storx/satellite/console/consoleauth"
	"storx/satellite/console/consoleweb"
	"storx/satellite/console/restkeys"
	"storx/satellite/console/userinfo"
	"storx/satellite/contact"
	"storx/satellite/gracefulexit"
	"storx/satellite/inspector"
	"storx/satellite/internalpb"
	"storx/satellite/mailservice"
	"storx/satellite/metabase"
	"storx/satellite/metainfo"
	"storx/satellite/metainfo/piecedeletion"
	"storx/satellite/nodestats"
	"storx/satellite/oidc"
	"storx/satellite/orders"
	"storx/satellite/overlay"
	"storx/satellite/payments"
	"storx/satellite/payments/storxscan"
	"storx/satellite/payments/stripecoinpayments"
	"storx/satellite/reputation"
	"storx/satellite/snopayouts"
)

// API is the satellite API process.
//
// architecture: Peer
type API struct {
	Log      *zap.Logger
	Identity *identity.FullIdentity
	DB       DB

	Servers  *lifecycle.Group
	Services *lifecycle.Group

	Dialer          rpc.Dialer
	Server          *server.Server
	ExternalAddress string

	Version struct {
		Chore   *checker.Chore
		Service *checker.Service
	}

	Debug struct {
		Listener net.Listener
		Server   *debug.Server
	}

	Contact struct {
		Service  *contact.Service
		Endpoint *contact.Endpoint
	}

	Overlay struct {
		DB      overlay.DB
		Service *overlay.Service
	}

	Reputation struct {
		Service *reputation.Service
	}

	Orders struct {
		DB       orders.DB
		Endpoint *orders.Endpoint
		Service  *orders.Service
		Chore    *orders.Chore
	}

	Metainfo struct {
		Metabase      *metabase.DB
		PieceDeletion *piecedeletion.Service
		Endpoint      *metainfo.Endpoint
	}

	Userinfo struct {
		Endpoint *userinfo.Endpoint
	}

	Inspector struct {
		Endpoint *inspector.Endpoint
	}

	Accounting struct {
		ProjectUsage *accounting.Service
	}

	LiveAccounting struct {
		Cache accounting.Cache
	}

	ProjectLimits struct {
		Cache *accounting.ProjectLimitCache
	}

	Mail struct {
		Service *mailservice.Service
	}

	Payments struct {
		Accounts       payments.Accounts
		DepositWallets payments.DepositWallets

		StorxscanService *storxscan.Service
		StorxscanClient  *storxscan.Client

		StripeService *stripecoinpayments.Service
		StripeClient  stripecoinpayments.StripeClient
	}

	REST struct {
		Keys *restkeys.Service
	}

	Console struct {
		Listener   net.Listener
		Service    *console.Service
		Endpoint   *consoleweb.Server
		AuthTokens *consoleauth.Service
	}

	NodeStats struct {
		Endpoint *nodestats.Endpoint
	}

	OIDC struct {
		Service *oidc.Service
	}

	SNOPayouts struct {
		Endpoint *snopayouts.Endpoint
		Service  *snopayouts.Service
		DB       snopayouts.DB
	}

	GracefulExit struct {
		Endpoint *gracefulexit.Endpoint
	}

	Analytics struct {
		Service *analytics.Service
	}

	ABTesting struct {
		Service *abtesting.Service
	}

	Buckets struct {
		Service *buckets.Service
	}
}

// NewAPI creates a new satellite API process.
func NewAPI(log *zap.Logger, full *identity.FullIdentity, db DB,
	metabaseDB *metabase.DB, revocationDB extensions.RevocationDB,
	liveAccounting accounting.Cache, rollupsWriteCache *orders.RollupsWriteCache,
	config *Config, versionInfo version.Info, atomicLogLevel *zap.AtomicLevel) (*API, error) {
	peer := &API{
		Log:             log,
		Identity:        full,
		DB:              db,
		ExternalAddress: config.Contact.ExternalAddress,

		Servers:  lifecycle.NewGroup(log.Named("servers")),
		Services: lifecycle.NewGroup(log.Named("services")),
	}

	{ // setup buckets service
		peer.Buckets.Service = buckets.NewService(db.Buckets(), metabaseDB)
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
		debugConfig.ControlTitle = "API"
		peer.Debug.Server = debug.NewServerWithAtomicLevel(log.Named("debug"), peer.Debug.Listener, monkit.Default, debugConfig, atomicLogLevel)
		peer.Servers.Add(lifecycle.Item{
			Name:  "debug",
			Run:   peer.Debug.Server.Run,
			Close: peer.Debug.Server.Close,
		})
	}

	var err error

	{
		peer.Log.Info("Version info",
			zap.Stringer("Version", versionInfo.Version.Version),
			zap.String("Commit Hash", versionInfo.CommitHash),
			zap.Stringer("Build Timestamp", versionInfo.Timestamp),
			zap.Bool("Release Build", versionInfo.Release),
		)

		peer.Version.Service = checker.NewService(log.Named("version"), config.Version, versionInfo, "Satellite")
		peer.Version.Chore = checker.NewChore(peer.Version.Service, config.Version.CheckInterval)

		peer.Services.Add(lifecycle.Item{
			Name: "version",
			Run:  peer.Version.Chore.Run,
		})
	}

	{ // setup listener and server
		sc := config.Server

		tlsOptions, err := tlsopts.NewOptions(peer.Identity, sc.Config, revocationDB)
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		peer.Dialer = rpc.NewDefaultDialer(tlsOptions)

		peer.Server, err = server.New(log.Named("server"), tlsOptions, sc)
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		if peer.ExternalAddress == "" {
			// not ideal, but better than nothing
			peer.ExternalAddress = peer.Server.Addr().String()
		}

		peer.Servers.Add(lifecycle.Item{
			Name: "server",
			Run: func(ctx context.Context) error {
				// Don't change the format of this comment, it is used to figure out the node id.
				peer.Log.Info(fmt.Sprintf("Node %s started", peer.Identity.ID))
				peer.Log.Info(fmt.Sprintf("Public server started on %s", peer.Addr()))
				peer.Log.Info(fmt.Sprintf("Private server started on %s", peer.PrivateAddr()))
				return peer.Server.Run(ctx)
			},
			Close: peer.Server.Close,
		})
	}

	{ // setup mailservice
		peer.Mail.Service, err = setupMailService(peer.Log, *config)
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		peer.Services.Add(lifecycle.Item{
			Name:  "mail:service",
			Close: peer.Mail.Service.Close,
		})
	}

	{ // setup overlay
		peer.Overlay.DB = peer.DB.OverlayCache()

		peer.Overlay.Service, err = overlay.NewService(peer.Log.Named("overlay"), peer.Overlay.DB, peer.DB.NodeEvents(), config.Console.ExternalAddress, config.Console.SatelliteName, config.Overlay)
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}
		peer.Services.Add(lifecycle.Item{
			Name:  "overlay",
			Run:   peer.Overlay.Service.Run,
			Close: peer.Overlay.Service.Close,
		})
	}

	{ // setup reputation
		reputationDB := peer.DB.Reputation()
		if config.Reputation.FlushInterval > 0 {
			cachingDB := reputation.NewCachingDB(log.Named("reputation:writecache"), reputationDB, config.Reputation)
			peer.Services.Add(lifecycle.Item{
				Name: "reputation:writecache",
				Run:  cachingDB.Manage,
			})
			reputationDB = cachingDB
		}
		peer.Reputation.Service = reputation.NewService(peer.Log.Named("reputation"), peer.Overlay.Service, reputationDB, config.Reputation)
		peer.Services.Add(lifecycle.Item{
			Name:  "reputation",
			Close: peer.Reputation.Service.Close,
		})
	}

	{ // setup contact service
		pbVersion, err := versionInfo.Proto()
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		self := &overlay.NodeDossier{
			Node: pb.Node{
				Id: peer.ID(),
				Address: &pb.NodeAddress{
					Address: peer.Addr(),
				},
			},
			Type:    pb.NodeType_SATELLITE,
			Version: *pbVersion,
		}
		peer.Contact.Service = contact.NewService(peer.Log.Named("contact:service"), self, peer.Overlay.Service, peer.DB.PeerIdentities(), peer.Dialer, config.Contact)
		peer.Contact.Endpoint = contact.NewEndpoint(peer.Log.Named("contact:endpoint"), peer.Contact.Service)
		if err := pb.DRPCRegisterNode(peer.Server.DRPC(), peer.Contact.Endpoint); err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		peer.Services.Add(lifecycle.Item{
			Name:  "contact:service",
			Close: peer.Contact.Service.Close,
		})
	}

	{ // setup live accounting
		peer.LiveAccounting.Cache = liveAccounting
	}

	{ // setup project limits
		peer.ProjectLimits.Cache = accounting.NewProjectLimitCache(peer.DB.ProjectAccounting(),
			config.Console.Config.UsageLimits.Storage.Free,
			config.Console.Config.UsageLimits.Bandwidth.Free,
			config.Console.Config.UsageLimits.Segment.Free,
			config.ProjectLimit,
		)
	}

	{ // setup accounting project usage
		peer.Accounting.ProjectUsage = accounting.NewService(
			peer.DB.ProjectAccounting(),
			peer.LiveAccounting.Cache,
			peer.ProjectLimits.Cache,
			*metabaseDB,
			config.LiveAccounting.BandwidthCacheTTL,
			config.LiveAccounting.AsOfSystemInterval,
		)
	}

	{ // setup oidc
		peer.OIDC.Service = oidc.NewService(db.OIDC())
	}

	{ // setup orders
		peer.Orders.DB = rollupsWriteCache
		peer.Orders.Chore = orders.NewChore(log.Named("orders:chore"), rollupsWriteCache, config.Orders)
		peer.Services.Add(lifecycle.Item{
			Name:  "orders:chore",
			Run:   peer.Orders.Chore.Run,
			Close: peer.Orders.Chore.Close,
		})
		peer.Debug.Server.Panel.Add(
			debug.Cycle("Orders Chore", peer.Orders.Chore.Loop))
		var err error
		peer.Orders.Service, err = orders.NewService(
			peer.Log.Named("orders:service"),
			signing.SignerFromFullIdentity(peer.Identity),
			peer.Overlay.Service,
			peer.Orders.DB,
			config.Orders,
		)
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		satelliteSignee := signing.SigneeFromPeerIdentity(peer.Identity.PeerIdentity())
		peer.Orders.Endpoint = orders.NewEndpoint(
			peer.Log.Named("orders:endpoint"),
			satelliteSignee,
			peer.Orders.DB,
			peer.DB.NodeAPIVersion(),
			config.Orders.OrdersSemaphoreSize,
			peer.Orders.Service,
		)

		if err := pb.DRPCRegisterOrders(peer.Server.DRPC(), peer.Orders.Endpoint); err != nil {
			return nil, errs.Combine(err, peer.Close())
		}
	}

	{ // setup analytics service
		peer.Analytics.Service = analytics.NewService(peer.Log.Named("analytics:service"), config.Analytics, config.Console.SatelliteName)

		peer.Services.Add(lifecycle.Item{
			Name:  "analytics:service",
			Run:   peer.Analytics.Service.Run,
			Close: peer.Analytics.Service.Close,
		})
	}

	{ // setup AB test service
		peer.ABTesting.Service = abtesting.NewService(peer.Log.Named("abtesting:service"), config.Console.ABTesting)

		peer.Services.Add(lifecycle.Item{
			Name: "abtesting:service",
		})
	}

	{ // setup metainfo
		peer.Metainfo.Metabase = metabaseDB

		peer.Metainfo.PieceDeletion, err = piecedeletion.NewService(
			peer.Log.Named("metainfo:piecedeletion"),
			peer.Dialer,
			// TODO use cache designed for deletion
			peer.Overlay.Service.DownloadSelectionCache,
			config.Metainfo.PieceDeletion,
		)
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}
		peer.Services.Add(lifecycle.Item{
			Name:  "metainfo:piecedeletion",
			Run:   peer.Metainfo.PieceDeletion.Run,
			Close: peer.Metainfo.PieceDeletion.Close,
		})

		peer.Metainfo.Endpoint, err = metainfo.NewEndpoint(
			peer.Log.Named("metainfo:endpoint"),
			peer.Buckets.Service,
			peer.Metainfo.Metabase,
			peer.Metainfo.PieceDeletion,
			peer.Orders.Service,
			peer.Overlay.Service,
			peer.DB.Attribution(),
			peer.DB.PeerIdentities(),
			peer.DB.Console().APIKeys(),
			peer.Accounting.ProjectUsage,
			peer.ProjectLimits.Cache,
			peer.DB.Console().Projects(),
			signing.SignerFromFullIdentity(peer.Identity),
			peer.DB.Revocation(),
			config.Metainfo,
		)
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		if err := pb.DRPCRegisterMetainfo(peer.Server.DRPC(), peer.Metainfo.Endpoint); err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		peer.Services.Add(lifecycle.Item{
			Name:  "metainfo:endpoint",
			Close: peer.Metainfo.Endpoint.Close,
		})
	}

	{ // setup userinfo.
		if config.Userinfo.Enabled {

			peer.Userinfo.Endpoint, err = userinfo.NewEndpoint(
				peer.Log.Named("userinfo:endpoint"),
				peer.DB.Console().Users(),
				peer.DB.Console().APIKeys(),
				peer.DB.Console().Projects(),
				config.Userinfo,
			)
			if err != nil {
				return nil, errs.Combine(err, peer.Close())
			}

			if err := pb.DRPCRegisterUserInfo(peer.Server.DRPC(), peer.Userinfo.Endpoint); err != nil {
				return nil, errs.Combine(err, peer.Close())
			}

			peer.Services.Add(lifecycle.Item{
				Name:  "userinfo:endpoint",
				Close: peer.Userinfo.Endpoint.Close,
			})
		} else {
			peer.Log.Named("userinfo:endpoint").Info("disabled")
		}
	}

	{ // setup inspector
		peer.Inspector.Endpoint = inspector.NewEndpoint(
			peer.Log.Named("inspector"),
			peer.Overlay.Service,
			peer.Metainfo.Metabase,
		)
		if err := internalpb.DRPCRegisterHealthInspector(peer.Server.PrivateDRPC(), peer.Inspector.Endpoint); err != nil {
			return nil, errs.Combine(err, peer.Close())
		}
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

		peer.Payments.StripeService, err = stripecoinpayments.NewService(
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

		peer.Payments.StripeClient = stripeClient
		peer.Payments.Accounts = peer.Payments.StripeService.Accounts()

		peer.Payments.StorxscanClient = storxscan.NewClient(
			pc.Storxscan.Endpoint,
			pc.Storxscan.Auth.Identifier,
			pc.Storxscan.Auth.Secret)

		peer.Payments.StorxscanService = storxscan.NewService(log.Named("storxscan-service"),
			peer.DB.Wallets(),
			peer.DB.StorxscanPayments(),
			peer.Payments.StorxscanClient)
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		peer.Payments.DepositWallets = peer.Payments.StorxscanService
	}

	{ // setup account management api keys
		peer.REST.Keys = restkeys.NewService(peer.DB.OIDC().OAuthTokens(), config.RESTKeys)
	}

	{ // setup console
		consoleConfig := config.Console
		peer.Console.Listener, err = net.Listen("tcp", consoleConfig.Address)
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}
		if consoleConfig.AuthTokenSecret == "" {
			return nil, errs.New("Auth token secret required")
		}

		peer.Console.AuthTokens = consoleauth.NewService(config.ConsoleAuth, &consoleauth.Hmac{Secret: []byte(consoleConfig.AuthTokenSecret)})

		externalAddress := consoleConfig.ExternalAddress
		if externalAddress == "" {
			externalAddress = "http://" + peer.Console.Listener.Addr().String()
		}

		peer.Console.Service, err = console.NewService(
			peer.Log.Named("console:service"),
			peer.DB.Console(),
			peer.REST.Keys,
			peer.DB.ProjectAccounting(),
			peer.Accounting.ProjectUsage,
			peer.Buckets.Service,
			peer.Payments.Accounts,
			peer.Payments.DepositWallets,
			peer.DB.Billing(),
			peer.Analytics.Service,
			peer.Console.AuthTokens,
			peer.Mail.Service,
			externalAddress,
			consoleConfig.Config,
		)
		if err != nil {
			return nil, errs.Combine(err, peer.Close())
		}

		accountFreezeService := console.NewAccountFreezeService(db.Console().AccountFreezeEvents(), db.Console().Users(), db.Console().Projects(), peer.Analytics.Service)

		peer.Console.Endpoint = consoleweb.NewServer(
			peer.Log.Named("console:endpoint"),
			consoleConfig,
			peer.Console.Service,
			peer.OIDC.Service,
			peer.Mail.Service,
			peer.Analytics.Service,
			peer.ABTesting.Service,
			accountFreezeService,
			peer.Console.Listener,
			config.Payments.StripeCoinPayments.StripePublicKey,
			peer.URL(),
			config.Payments.PackagePlans,
		)

		peer.Servers.Add(lifecycle.Item{
			Name:  "console:endpoint",
			Run:   peer.Console.Endpoint.Run,
			Close: peer.Console.Endpoint.Close,
		})
	}

	{ // setup node stats endpoint
		peer.NodeStats.Endpoint = nodestats.NewEndpoint(
			peer.Log.Named("nodestats:endpoint"),
			peer.Overlay.DB,
			peer.Reputation.Service,
			peer.DB.StoragenodeAccounting(),
			config.Payments,
		)
		if err := pb.DRPCRegisterNodeStats(peer.Server.DRPC(), peer.NodeStats.Endpoint); err != nil {
			return nil, errs.Combine(err, peer.Close())
		}
	}

	{ // setup SnoPayout endpoint
		peer.SNOPayouts.DB = peer.DB.SNOPayouts()
		peer.SNOPayouts.Service = snopayouts.NewService(
			peer.Log.Named("payouts:service"),
			peer.SNOPayouts.DB)
		peer.SNOPayouts.Endpoint = snopayouts.NewEndpoint(
			peer.Log.Named("payouts:endpoint"),
			peer.DB.StoragenodeAccounting(),
			peer.Overlay.DB,
			peer.SNOPayouts.Service)
		if err := pb.DRPCRegisterHeldAmount(peer.Server.DRPC(), peer.SNOPayouts.Endpoint); err != nil {
			return nil, errs.Combine(err, peer.Close())
		}
	}

	{ // setup graceful exit
		if config.GracefulExit.Enabled {
			peer.GracefulExit.Endpoint = gracefulexit.NewEndpoint(
				peer.Log.Named("gracefulexit:endpoint"),
				signing.SignerFromFullIdentity(peer.Identity),
				peer.DB.GracefulExit(),
				peer.Overlay.DB,
				peer.Overlay.Service,
				peer.Reputation.Service,
				peer.Metainfo.Metabase,
				peer.Orders.Service,
				peer.DB.PeerIdentities(),
				config.GracefulExit)

			if err := pb.DRPCRegisterSatelliteGracefulExit(peer.Server.DRPC(), peer.GracefulExit.Endpoint); err != nil {
				return nil, errs.Combine(err, peer.Close())
			}
		} else {
			peer.Log.Named("gracefulexit").Info("disabled")
		}
	}

	return peer, nil
}

// Run runs satellite until it's either closed or it errors.
func (peer *API) Run(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)

	group, ctx := errgroup.WithContext(ctx)

	pprof.Do(ctx, pprof.Labels("subsystem", "api"), func(ctx context.Context) {
		peer.Servers.Run(ctx, group)
		peer.Services.Run(ctx, group)

		pprof.Do(ctx, pprof.Labels("name", "subsystem-wait"), func(ctx context.Context) {
			err = group.Wait()
		})
	})
	return err
}

// Close closes all the resources.
func (peer *API) Close() error {
	return errs.Combine(
		peer.Servers.Close(),
		peer.Services.Close(),
	)
}

// ID returns the peer ID.
func (peer *API) ID() storx.NodeID { return peer.Identity.ID }

// Addr returns the public address.
func (peer *API) Addr() string {
	return peer.ExternalAddress
}

// URL returns the storx.NodeURL.
func (peer *API) URL() storx.NodeURL {
	return storx.NodeURL{ID: peer.ID(), Address: peer.Addr()}
}

// PrivateAddr returns the private address.
func (peer *API) PrivateAddr() string { return peer.Server.PrivateAddr().String() }
