// Copyright (C) 2023 Storx Labs, Inc.
// See LICENSE for copying information.

package accountfreeze_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"

	"common/testcontext"
	"storx/private/testplanet"
	"storx/satellite"
	"storx/satellite/console"
	"storx/satellite/payments/stripecoinpayments"
)

func TestAutoFreezeChore(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 0, UplinkCount: 0,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.AccountFreeze.Enabled = true
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		sat := planet.Satellites[0]
		stripeClient := sat.API.Payments.StripeClient
		invoicesDB := sat.Core.Payments.Accounts.Invoices()
		customerDB := sat.Core.DB.StripeCoinPayments().Customers()
		usersDB := sat.DB.Console().Users()
		projectsDB := sat.DB.Console().Projects()
		service := console.NewAccountFreezeService(sat.DB.Console().AccountFreezeEvents(), usersDB, projectsDB, sat.API.Analytics.Service)
		chore := sat.Core.Payments.AccountFreeze

		user, err := sat.AddUser(ctx, console.CreateUser{
			FullName: "Test User",
			Email:    "user@mail.test",
		}, 1)
		require.NoError(t, err)

		cus1, err := customerDB.GetCustomerID(ctx, user.ID)
		require.NoError(t, err)

		amount := int64(100)
		curr := string(stripe.CurrencyUSD)

		t.Run("No freeze event for paid invoice", func(t *testing.T) {
			item, err := stripeClient.InvoiceItems().New(&stripe.InvoiceItemParams{
				Params:   stripe.Params{Context: ctx},
				Amount:   &amount,
				Currency: &curr,
				Customer: &cus1,
			})
			require.NoError(t, err)

			items := make([]*stripe.InvoiceUpcomingInvoiceItemParams, 0, 1)
			items = append(items, &stripe.InvoiceUpcomingInvoiceItemParams{
				InvoiceItem: &item.ID,
				Amount:      &amount,
				Currency:    &curr,
			})
			inv, err := stripeClient.Invoices().New(&stripe.InvoiceParams{
				Params:       stripe.Params{Context: ctx},
				Customer:     &cus1,
				InvoiceItems: items,
			})
			require.NoError(t, err)

			inv, err = stripeClient.Invoices().Pay(inv.ID, &stripe.InvoicePayParams{
				Params: stripe.Params{Context: ctx},
			})
			require.NoError(t, err)
			require.Equal(t, stripe.InvoiceStatusPaid, inv.Status)

			failed, err := invoicesDB.ListFailed(ctx)
			require.NoError(t, err)
			require.Equal(t, 0, len(failed))

			chore.Loop.TriggerWait()

			// user should not be warned or frozen.
			freeze, warning, err := service.GetAll(ctx, user.ID)
			require.NoError(t, err)
			require.Nil(t, warning)
			require.Nil(t, freeze)

			// forward date to after the grace period
			chore.TestSetNow(func() time.Time {
				return time.Now().AddDate(0, 0, 50)
			})
			chore.Loop.TriggerWait()

			// user should still not be warned or frozen.
			freeze, warning, err = service.GetAll(ctx, user.ID)
			require.NoError(t, err)
			require.Nil(t, freeze)
			require.Nil(t, warning)
		})

		t.Run("Freeze event for failed invoice", func(t *testing.T) {
			// reset chore clock
			chore.TestSetNow(time.Now)

			item, err := stripeClient.InvoiceItems().New(&stripe.InvoiceItemParams{
				Params:   stripe.Params{Context: ctx},
				Amount:   &amount,
				Currency: &curr,
				Customer: &cus1,
			})
			require.NoError(t, err)

			items := make([]*stripe.InvoiceUpcomingInvoiceItemParams, 0, 1)
			items = append(items, &stripe.InvoiceUpcomingInvoiceItemParams{
				InvoiceItem: &item.ID,
				Amount:      &amount,
				Currency:    &curr,
			})
			inv, err := stripeClient.Invoices().New(&stripe.InvoiceParams{
				Params:       stripe.Params{Context: ctx},
				Customer:     &cus1,
				InvoiceItems: items,
			})
			require.NoError(t, err)

			paymentMethod := stripecoinpayments.MockInvoicesPayFailure
			inv, err = stripeClient.Invoices().Pay(inv.ID, &stripe.InvoicePayParams{
				Params:        stripe.Params{Context: ctx},
				PaymentMethod: &paymentMethod,
			})
			require.Error(t, err)
			require.Equal(t, stripe.InvoiceStatusOpen, inv.Status)

			failed, err := invoicesDB.ListFailed(ctx)
			require.NoError(t, err)
			require.Equal(t, 1, len(failed))
			require.Equal(t, inv.ID, failed[0].ID)

			chore.Loop.TriggerWait()

			// user should be warned the first time
			freeze, warning, err := service.GetAll(ctx, user.ID)
			require.NoError(t, err)
			require.NotNil(t, warning)
			require.Nil(t, freeze)

			chore.TestSetNow(func() time.Time {
				// current date is now after grace period
				return time.Now().AddDate(0, 0, 50)
			})
			chore.Loop.TriggerWait()

			// user should be frozen this time around
			freeze, _, err = service.GetAll(ctx, user.ID)
			require.NoError(t, err)
			require.NotNil(t, freeze)
		})
	})
}
