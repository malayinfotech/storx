// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package storxscan_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zeebo/errs"

	"common/currency"
	"common/testcontext"
	"storx/satellite/payments/storxscan"
	"storx/satellite/payments/storxscan/blockchaintest"
	"storx/satellite/payments/storxscan/storxscantest"
)

func TestClientMocked(t *testing.T) {
	ctx := testcontext.New(t)
	now := time.Now().Round(time.Second).UTC()

	var payments []storxscan.Payment
	for i := 0; i < 100; i++ {
		payments = append(payments, storxscan.Payment{
			From:        blockchaintest.NewAddress(),
			To:          blockchaintest.NewAddress(),
			TokenValue:  currency.AmountFromBaseUnits(int64(i)*100000000, currency.StorxToken),
			USDValue:    currency.AmountFromBaseUnits(int64(i)*1100000, currency.USDollarsMicro),
			BlockHash:   blockchaintest.NewHash(),
			BlockNumber: int64(i),
			Transaction: blockchaintest.NewHash(),
			LogIndex:    i,
			Timestamp:   now.Add(time.Duration(i) * time.Second),
		})
	}
	latestBlock := storxscan.Header{
		Hash:      payments[len(payments)-1].BlockHash,
		Number:    payments[len(payments)-1].BlockNumber,
		Timestamp: payments[len(payments)-1].Timestamp,
	}

	const (
		identifier = "eu"
		secret     = "secret"
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		if err := storxscantest.CheckAuth(r, identifier, secret); err != nil {
			storxscantest.ServeJSONError(t, w, http.StatusUnauthorized, err)
			return
		}

		var from int64
		if s := r.URL.Query().Get("from"); s != "" {
			from, err = strconv.ParseInt(s, 10, 64)
			if err != nil {
				storxscantest.ServeJSONError(t, w, http.StatusBadRequest, errs.New("from parameter is missing"))
				return
			}
		}

		storxscantest.ServePayments(t, w, from, latestBlock, payments)
	}))
	defer server.Close()

	client := storxscan.NewClient(server.URL, "eu", "secret")

	t.Run("all payments from 0", func(t *testing.T) {
		actual, err := client.Payments(ctx, 0)
		require.NoError(t, err)
		require.Equal(t, latestBlock, actual.LatestBlock)
		require.Equal(t, len(payments), len(actual.Payments))
		require.Equal(t, payments, actual.Payments)
	})
	t.Run("payments from 50", func(t *testing.T) {
		actual, err := client.Payments(ctx, 50)
		require.NoError(t, err)
		require.Equal(t, latestBlock, actual.LatestBlock)
		require.Equal(t, 50, len(actual.Payments))
		require.Equal(t, payments[50:], actual.Payments)
	})
}

func TestClientMockedUnauthorized(t *testing.T) {
	ctx := testcontext.New(t)

	const (
		identifier = "eu"
		secret     = "secret"
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := storxscantest.CheckAuth(r, identifier, secret); err != nil {
			storxscantest.ServeJSONError(t, w, http.StatusUnauthorized, err)
			return
		}
	}))
	defer server.Close()

	t.Run("empty credentials", func(t *testing.T) {
		client := storxscan.NewClient(server.URL, "", "")
		_, err := client.Payments(ctx, 0)
		require.Error(t, err)
		require.True(t, storxscan.ClientErrUnauthorized.Has(err))
		require.Equal(t, "identifier is invalid", errs.Unwrap(err).Error())
	})

	t.Run("invalid identifier", func(t *testing.T) {
		client := storxscan.NewClient(server.URL, "invalid", "secret")
		_, err := client.Payments(ctx, 0)
		require.Error(t, err)
		require.True(t, storxscan.ClientErrUnauthorized.Has(err))
		require.Equal(t, "identifier is invalid", errs.Unwrap(err).Error())
	})

	t.Run("invalid secret", func(t *testing.T) {
		client := storxscan.NewClient(server.URL, "eu", "invalid")
		_, err := client.Payments(ctx, 0)
		require.Error(t, err)
		require.True(t, storxscan.ClientErrUnauthorized.Has(err))
		require.Equal(t, "secret is invalid", errs.Unwrap(err).Error())
	})
}
