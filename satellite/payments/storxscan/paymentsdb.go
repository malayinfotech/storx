// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package storxscan

import (
	"context"
	"time"

	"github.com/zeebo/errs"

	"common/currency"
	"storx/private/blockchain"
	"storx/satellite/payments"
)

// ErrNoPayments represents err when there is no payments in the DB.
var ErrNoPayments = errs.New("no payments in the database")

// PaymentsDB is storxscan payments DB interface.
//
// architecture: Database
type PaymentsDB interface {
	// InsertBatch inserts list of payments into DB.
	InsertBatch(ctx context.Context, payments []CachedPayment) error
	// List returns list of all storxscan payments order by block number and log index desc mainly for testing.
	List(ctx context.Context) ([]CachedPayment, error)
	// ListWallet returns list of storxscan payments order by block number and log index desc.
	ListWallet(ctx context.Context, wallet blockchain.Address, limit int, offset int64) ([]CachedPayment, error)
	// LastBlock returns the highest block known to DB for specified payment status.
	LastBlock(ctx context.Context, status payments.PaymentStatus) (int64, error)
	// DeletePending removes all pending transactions from the DB.
	DeletePending(ctx context.Context) error
	// ListConfirmed returns list of confirmed storxscan payments greater than the given timestamp.
	ListConfirmed(ctx context.Context, blockNumber int64, logIndex int) ([]CachedPayment, error)
}

// CachedPayment holds cached data of storxscan payment.
type CachedPayment struct {
	From        blockchain.Address     `json:"from"`
	To          blockchain.Address     `json:"to"`
	TokenValue  currency.Amount        `json:"tokenValue"`
	USDValue    currency.Amount        `json:"usdValue"`
	Status      payments.PaymentStatus `json:"status"`
	BlockHash   blockchain.Hash        `json:"blockHash"`
	BlockNumber int64                  `json:"blockNumber"`
	Transaction blockchain.Hash        `json:"transaction"`
	LogIndex    int                    `json:"logIndex"`
	Timestamp   time.Time              `json:"timestamp"`
}
