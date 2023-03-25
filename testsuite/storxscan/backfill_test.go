// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package storxscan

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"common/currency"
	"common/testcontext"
	blockchain2 "storx/private/blockchain"
	"storx/private/testplanet"
	"storx/satellite/payments"
	"storx/testsuite/storxscan/storxscantest"
	"storxscan/blockchain"
	"storxscan/private/testeth/testtoken"
)

func TestBackfillPayments(t *testing.T) {
	storxscantest.Run(t, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet, stack *storxscantest.Stack) {
		receiver, _ := blockchain.AddressFromHex("0x27e3d303B0B70B1b17f14525b48Ae7c45D34666f")
		err := stack.App.Wallets.Service.Register(ctx, "eu", map[blockchain.Address]string{
			receiver: "test",
		})
		require.NoError(t, err)

		sat := planet.Satellites[0]
		sat.Core.Payments.StorxscanChore.TransactionCycle.Pause()

		// claim wallet
		_, err = sat.API.Payments.StorxscanClient.ClaimNewEthAddress(ctx)
		require.NoError(t, err)

		client := stack.Network.Dial()
		defer client.Close()
		accs := stack.Network.Accounts()

		tk, err := testtoken.NewTestToken(blockchain.Address(stack.Token), client)
		require.NoError(t, err)

		opts := stack.Network.TransactOptions(ctx, accs[0], 1)
		tx, err := tk.Transfer(opts, receiver, big.NewInt(10000))
		require.NoError(t, err)
		rcpt, err := stack.Network.WaitForTx(ctx, tx.Hash())
		require.NoError(t, err)

		block, err := client.BlockByNumber(ctx, rcpt.BlockNumber)
		require.NoError(t, err)
		blockTime := time.Unix(int64(block.Time()), 0)

		sat.Core.Payments.StorxscanChore.TransactionCycle.TriggerWait()

		pmnts, err := sat.API.Payments.StorxscanService.Payments(ctx, blockchain2.Address(receiver), 1, 0)
		require.NoError(t, err)
		require.Len(t, pmnts, 1)

		expected := payments.WalletPayment{
			From:        blockchain2.Address(accs[0].Address),
			To:          blockchain2.Address(receiver),
			TokenValue:  currency.AmountFromBaseUnits(10000, currency.StorxToken),
			USDValue:    currency.AmountFromBaseUnits(100, currency.USDollarsMicro),
			Status:      payments.PaymentStatusPending,
			BlockHash:   blockchain2.Hash(block.Hash()),
			BlockNumber: block.Number().Int64(),
			Transaction: blockchain2.Hash(rcpt.TxHash),
			LogIndex:    0,
			Timestamp:   blockTime.UTC(),
		}
		require.Equal(t, expected, pmnts[0])

		// disable
		err = stack.CloseApp()
		require.NoError(t, err)

		// send 2nd tx
		opts = stack.Network.TransactOptions(ctx, accs[0], 2)
		tx, err = tk.Transfer(opts, receiver, big.NewInt(10000))
		require.NoError(t, err)
		rcpt, err = stack.Network.WaitForTx(ctx, tx.Hash())
		require.NoError(t, err)

		block, err = client.BlockByNumber(ctx, rcpt.BlockNumber)
		require.NoError(t, err)
		blockTime = time.Unix(int64(block.Time()), 0)

		sat.Core.Payments.StorxscanChore.TransactionCycle.TriggerWait()

		// ensure there is still only one pmnt in the db
		pmnts, err = sat.API.Payments.StorxscanService.Payments(ctx, blockchain2.Address(receiver), 2, 0)
		require.NoError(t, err)
		require.Len(t, pmnts, 1)
		require.Equal(t, expected, pmnts[0])

		// Start storxscan
		err = stack.StartApp()
		require.NoError(t, err)

		sat.Core.Payments.StorxscanChore.TransactionCycle.TriggerWait()

		pmnts, err = sat.API.Payments.StorxscanService.Payments(ctx, blockchain2.Address(receiver), 2, 0)
		require.NoError(t, err)
		require.Len(t, pmnts, 2)

		expected = payments.WalletPayment{
			From:        blockchain2.Address(accs[0].Address),
			To:          blockchain2.Address(receiver),
			TokenValue:  currency.AmountFromBaseUnits(10000, currency.StorxToken),
			USDValue:    currency.AmountFromBaseUnits(100, currency.USDollarsMicro),
			Status:      payments.PaymentStatusPending,
			BlockHash:   blockchain2.Hash(block.Hash()),
			BlockNumber: block.Number().Int64(),
			Transaction: blockchain2.Hash(rcpt.TxHash),
			LogIndex:    0,
			Timestamp:   blockTime.UTC(),
		}
		require.Equal(t, expected, pmnts[0])
	})
}
