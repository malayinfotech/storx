// Copyright (C) 2023 Storx Labs, Inc.
// See LICENSE for copying information.

package accountfreeze

import (
	"context"
	"time"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"common/sync2"
	"storx/satellite/analytics"
	"storx/satellite/console"
	"storx/satellite/payments"
	"storx/satellite/payments/stripecoinpayments"
)

var (
	// Error is the standard error class for automatic freeze errors.
	Error = errs.Class("account-freeze-chore")
	mon   = monkit.Package()
)

// Config contains configurable values for account freeze chore.
type Config struct {
	Enabled        bool          `help:"whether to run this chore." default:"false"`
	Interval       time.Duration `help:"How often to run this chore, which is how often unpaid invoices are checked." default:"24h"`
	GracePeriod    time.Duration `help:"How long to wait between a warning event and freezing an account." default:"720h"`
	PriceThreshold int64         `help:"The failed invoice amount beyond which an account will not be frozen" default:"2000"`
}

// Chore is a chore that checks for unpaid invoices and potentially freezes corresponding accounts.
type Chore struct {
	log           *zap.Logger
	freezeService *console.AccountFreezeService
	analytics     *analytics.Service
	usersDB       console.Users
	payments      payments.Accounts
	accounts      stripecoinpayments.DB
	config        Config
	nowFn         func() time.Time
	Loop          *sync2.Cycle
}

// NewChore is a constructor for Chore.
func NewChore(log *zap.Logger, accounts stripecoinpayments.DB, payments payments.Accounts, usersDB console.Users, freezeService *console.AccountFreezeService, analytics *analytics.Service, config Config) *Chore {
	return &Chore{
		log:           log,
		freezeService: freezeService,
		analytics:     analytics,
		usersDB:       usersDB,
		accounts:      accounts,
		config:        config,
		payments:      payments,
		nowFn:         time.Now,
		Loop:          sync2.NewCycle(config.Interval),
	}
}

// Run runs the chore.
func (chore *Chore) Run(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)
	return chore.Loop.Run(ctx, func(ctx context.Context) (err error) {

		invoices, err := chore.payments.Invoices().ListFailed(ctx)
		if err != nil {
			chore.log.Error("Could not list invoices", zap.Error(Error.Wrap(err)))
			return nil
		}

		for _, invoice := range invoices {
			userID, err := chore.accounts.Customers().GetUserID(ctx, invoice.CustomerID)
			if err != nil {
				chore.log.Error("Could not get userID", zap.String("invoice", invoice.ID), zap.Error(Error.Wrap(err)))
				continue
			}

			user, err := chore.usersDB.Get(ctx, userID)
			if err != nil {
				chore.log.Error("Could not get user", zap.String("invoice", invoice.ID), zap.Error(Error.Wrap(err)))
				continue
			}

			if invoice.Amount > chore.config.PriceThreshold {
				chore.analytics.TrackLargeUnpaidInvoice(invoice.ID, userID, user.Email)
				continue
			}

			freeze, warning, err := chore.freezeService.GetAll(ctx, userID)
			if err != nil {
				chore.log.Error("Could not check freeze status", zap.String("invoice", invoice.ID), zap.Error(Error.Wrap(err)))
				continue
			}
			if freeze != nil {
				// account already frozen
				continue
			}

			if warning == nil {
				err = chore.freezeService.WarnUser(ctx, userID)
				if err != nil {
					chore.log.Error("Could not add warning event", zap.String("invoice", invoice.ID), zap.Error(Error.Wrap(err)))
					continue
				}
				continue
			}

			if chore.nowFn().Sub(warning.CreatedAt) > chore.config.GracePeriod {
				err = chore.freezeService.FreezeUser(ctx, userID)
				if err != nil {
					chore.log.Error("Could not freeze account", zap.String("invoice", invoice.ID), zap.Error(Error.Wrap(err)))
					continue
				}
			}
		}

		return nil
	})
}

// TestSetNow sets nowFn on chore for testing.
func (chore *Chore) TestSetNow(f func() time.Time) {
	chore.nowFn = f
}

// Close closes the chore.
func (chore *Chore) Close() error {
	chore.Loop.Close()
	return nil
}
