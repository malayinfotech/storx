// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package satellite

import (
	"context"
	"net"
	"net/mail"
	"net/smtp"

	hw "github.com/jtolds/monkit-hw/v2"
	"github.com/spacemonkeygo/monkit/v3"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"common/identity"
	"private/debug"
	"private/tagsql"
	"storx/private/migrate"
	"storx/private/post"
	"storx/private/post/oauth2"
	"storx/private/server"
	version_checker "storx/private/version/checker"
	"storx/satellite/accounting"
	"storx/satellite/accounting/live"
	"storx/satellite/accounting/projectbwcleanup"
	"storx/satellite/accounting/rollup"
	"storx/satellite/accounting/rolluparchive"
	"storx/satellite/accounting/tally"
	"storx/satellite/admin"
	"storx/satellite/analytics"
	"storx/satellite/attribution"
	"storx/satellite/audit"
	"storx/satellite/buckets"
	"storx/satellite/compensation"
	"storx/satellite/console"
	"storx/satellite/console/consoleauth"
	"storx/satellite/console/consoleweb"
	"storx/satellite/console/emailreminders"
	"storx/satellite/console/restkeys"
	"storx/satellite/console/userinfo"
	"storx/satellite/contact"
	"storx/satellite/gc/bloomfilter"
	"storx/satellite/gc/sender"
	"storx/satellite/gracefulexit"
	"storx/satellite/mailservice"
	"storx/satellite/mailservice/simulate"
	"storx/satellite/metabase/rangedloop"
	"storx/satellite/metabase/zombiedeletion"
	"storx/satellite/metainfo"
	"storx/satellite/metainfo/expireddeletion"
	"storx/satellite/metrics"
	"storx/satellite/nodeapiversion"
	"storx/satellite/nodeevents"
	"storx/satellite/oidc"
	"storx/satellite/orders"
	"storx/satellite/overlay"
	"storx/satellite/overlay/offlinenodes"
	"storx/satellite/overlay/straynodes"
	"storx/satellite/payments/accountfreeze"
	"storx/satellite/payments/billing"
	"storx/satellite/payments/paymentsconfig"
	"storx/satellite/payments/storxscan"
	"storx/satellite/payments/stripecoinpayments"
	"storx/satellite/repair/checker"
	"storx/satellite/repair/queue"
	"storx/satellite/repair/repairer"
	"storx/satellite/reputation"
	"storx/satellite/revocation"
	"storx/satellite/snopayouts"
)

var mon = monkit.Package()

func init() {
	hw.Register(monkit.Default)
}

// DB is the master database for the satellite.
//
// architecture: Master Database
type DB interface {
	// MigrateToLatest initializes the database
	MigrateToLatest(ctx context.Context) error
	// CheckVersion checks the database is the correct version
	CheckVersion(ctx context.Context) error
	// Close closes the database
	Close() error

	// PeerIdentities returns a storage for peer identities
	PeerIdentities() overlay.PeerIdentities
	// OverlayCache returns database for caching overlay information
	OverlayCache() overlay.DB
	// NodeEvents returns a database for node event information
	NodeEvents() nodeevents.DB
	// Reputation returns database for audit reputation information
	Reputation() reputation.DB
	// Attribution returns database for partner keys information
	Attribution() attribution.DB
	// StoragenodeAccounting returns database for storing information about storagenode use
	StoragenodeAccounting() accounting.StoragenodeAccounting
	// ProjectAccounting returns database for storing information about project data use
	ProjectAccounting() accounting.ProjectAccounting
	// RepairQueue returns queue for segments that need repairing
	RepairQueue() queue.RepairQueue
	// VerifyQueue returns queue for segments chosen for verification
	VerifyQueue() audit.VerifyQueue
	// ReverifyQueue returns queue for pieces that need audit reverification
	ReverifyQueue() audit.ReverifyQueue
	// Console returns database for satellite console
	Console() console.DB
	// OIDC returns the database for OIDC resources.
	OIDC() oidc.DB
	// Orders returns database for orders
	Orders() orders.DB
	// Containment returns database for containment
	Containment() audit.Containment
	// Buckets returns the database to interact with buckets
	Buckets() buckets.DB
	// GracefulExit returns database for graceful exit
	GracefulExit() gracefulexit.DB
	// StripeCoinPayments returns stripecoinpayments database.
	StripeCoinPayments() stripecoinpayments.DB
	// Billing returns storxscan transactions database.
	Billing() billing.TransactionsDB
	// Wallets returns storxscan wallets database.
	Wallets() storxscan.WalletsDB
	// SNOPayouts returns database for payouts.
	SNOPayouts() snopayouts.DB
	// Compensation tracks storage node compensation
	Compensation() compensation.DB
	// Revocation tracks revoked macaroons
	Revocation() revocation.DB
	// NodeAPIVersion tracks nodes observed api usage
	NodeAPIVersion() nodeapiversion.DB
	// StorxscanPayments stores payments retrieved from storxscan.
	StorxscanPayments() storxscan.PaymentsDB

	// Testing provides access to testing facilities. These should not be used in production code.
	Testing() TestingDB
}

// TestingDB defines access to database testing facilities.
type TestingDB interface {
	// RawDB returns the underlying database connection to the primary database.
	RawDB() tagsql.DB
	// Schema returns the full schema for the database.
	Schema() string
	// TestMigrateToLatest initializes the database for testplanet.
	TestMigrateToLatest(ctx context.Context) error
	// ProductionMigration returns the primary migration.
	ProductionMigration() *migrate.Migration
	// TestMigration returns the migration used for tests.
	TestMigration() *migrate.Migration
}

// Config is the global config satellite.
type Config struct {
	Identity identity.Config
	Server   server.Config
	Debug    debug.Config

	Admin admin.Config

	Contact      contact.Config
	Overlay      overlay.Config
	OfflineNodes offlinenodes.Config
	NodeEvents   nodeevents.Config
	StrayNodes   straynodes.Config

	Metainfo metainfo.Config
	Orders   orders.Config

	Userinfo userinfo.Config

	Reputation reputation.Config

	Checker  checker.Config
	Repairer repairer.Config
	Audit    audit.Config

	GarbageCollection   sender.Config
	GarbageCollectionBF bloomfilter.Config

	RangedLoop rangedloop.Config

	ExpiredDeletion expireddeletion.Config
	ZombieDeletion  zombiedeletion.Config

	Tally            tally.Config
	Rollup           rollup.Config
	RollupArchive    rolluparchive.Config
	LiveAccounting   live.Config
	ProjectBWCleanup projectbwcleanup.Config

	Mail mailservice.Config

	Payments paymentsconfig.Config

	RESTKeys       restkeys.Config
	Console        consoleweb.Config
	ConsoleAuth    consoleauth.Config
	EmailReminders emailreminders.Config

	AccountFreeze accountfreeze.Config

	Version version_checker.Config

	GracefulExit gracefulexit.Config

	Metrics metrics.Config

	Compensation compensation.Config

	ProjectLimit accounting.ProjectLimitConfig

	Analytics analytics.Config
}

func setupMailService(log *zap.Logger, config Config) (*mailservice.Service, error) {
	// TODO(yar): test multiple satellites using same OAUTH credentials
	mailConfig := config.Mail

	// validate from mail address
	from, err := mail.ParseAddress(mailConfig.From)
	if err != nil {
		return nil, errs.New("SMTP from address '%s' couldn't be parsed: %v", mailConfig.From, err)
	}

	// validate smtp server address
	host, _, err := net.SplitHostPort(mailConfig.SMTPServerAddress)
	if err != nil && mailConfig.AuthType != "simulate" && mailConfig.AuthType != "nologin" {
		return nil, errs.New("SMTP server address '%s' couldn't be parsed: %v", mailConfig.SMTPServerAddress, err)
	}

	var sender mailservice.Sender
	switch mailConfig.AuthType {
	case "oauth2":
		creds := oauth2.Credentials{
			ClientID:     mailConfig.ClientID,
			ClientSecret: mailConfig.ClientSecret,
			TokenURI:     mailConfig.TokenURI,
		}
		token, err := oauth2.RefreshToken(context.TODO(), creds, mailConfig.RefreshToken)
		if err != nil {
			return nil, err
		}

		sender = &post.SMTPSender{
			From: *from,
			Auth: &oauth2.Auth{
				UserEmail: from.Address,
				Storage:   oauth2.NewTokenStore(creds, *token),
			},
			ServerAddress: mailConfig.SMTPServerAddress,
		}
	case "plain":
		sender = &post.SMTPSender{
			From:          *from,
			Auth:          smtp.PlainAuth("", mailConfig.Login, mailConfig.Password, host),
			ServerAddress: mailConfig.SMTPServerAddress,
		}
	case "login":
		sender = &post.SMTPSender{
			From: *from,
			Auth: post.LoginAuth{
				Username: mailConfig.Login,
				Password: mailConfig.Password,
			},
			ServerAddress: mailConfig.SMTPServerAddress,
		}
	case "nomail":
		sender = simulate.NoMail{}
	default:
		sender = simulate.NewDefaultLinkClicker(log.Named("mail:linkclicker"))
	}

	return mailservice.New(
		log.Named("mail:service"),
		sender,
		mailConfig.TemplatePath,
	)
}
