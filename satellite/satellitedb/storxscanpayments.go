// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package satellitedb

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeebo/errs"

	"common/currency"
	"private/dbutil/pgutil"
	"storx/private/blockchain"
	"storx/satellite/payments"
	"storx/satellite/payments/storxscan"
	"storx/satellite/satellitedb/dbx"
)

var _ storxscan.PaymentsDB = (*storxscanPayments)(nil)

// storxscanPayments implements storxscan.DB.
type storxscanPayments struct {
	db *satelliteDB
}

// InsertBatch inserts list of payments in a single transaction.
func (storxscanPayments *storxscanPayments) InsertBatch(ctx context.Context, payments []storxscan.CachedPayment) (err error) {
	defer mon.Task()(&ctx)(&err)

	cmnd := `INSERT INTO storxscan_payments(
				block_hash,
				block_number,
				transaction,
				log_index,
				from_address,
				to_address,
				token_value,
				usd_value,
				status,
				timestamp,
				created_at
			) SELECT
				UNNEST($1::BYTEA[]),
				UNNEST($2::INT8[]),
				UNNEST($3::BYTEA[]),
				UNNEST($4::INT4[]),
				UNNEST($5::BYTEA[]),
				UNNEST($6::BYTEA[]),
				UNNEST($7::INT8[]),
				UNNEST($8::INT8[]),
				UNNEST($9::TEXT[]),
				UNNEST($10::TIMESTAMPTZ[]),
				$11
			`
	var (
		blockHashes   = make([][]byte, 0, len(payments))
		blockNumbers  = make([]int64, 0, len(payments))
		transactions  = make([][]byte, 0, len(payments))
		logIndexes    = make([]int32, 0, len(payments))
		fromAddresses = make([][]byte, 0, len(payments))
		toAddresses   = make([][]byte, 0, len(payments))
		tokenValues   = make([]int64, 0, len(payments))
		usdValues     = make([]int64, 0, len(payments))
		statuses      = make([]string, 0, len(payments))
		timestamps    = make([]time.Time, 0, len(payments))

		createdAt = time.Now()
	)
	for i := range payments {
		payment := payments[i]
		blockHashes = append(blockHashes, payment.BlockHash[:])
		blockNumbers = append(blockNumbers, payment.BlockNumber)
		transactions = append(transactions, payment.Transaction[:])
		logIndexes = append(logIndexes, int32(payment.LogIndex))
		fromAddresses = append(fromAddresses, payment.From[:])
		toAddresses = append(toAddresses, payment.To[:])
		tokenValues = append(tokenValues, payment.TokenValue.BaseUnits())
		usdValues = append(usdValues, payment.USDValue.BaseUnits())
		statuses = append(statuses, string(payment.Status))
		timestamps = append(timestamps, payment.Timestamp)
	}

	_, err = storxscanPayments.db.ExecContext(ctx, cmnd,
		pgutil.ByteaArray(blockHashes),
		pgutil.Int8Array(blockNumbers),
		pgutil.ByteaArray(transactions),
		pgutil.Int4Array(logIndexes),
		pgutil.ByteaArray(fromAddresses),
		pgutil.ByteaArray(toAddresses),
		pgutil.Int8Array(tokenValues),
		pgutil.Int8Array(usdValues),
		pgutil.TextArray(statuses),
		pgutil.TimestampTZArray(timestamps),
		createdAt)
	return err
}

// List returns list of storxscan payments order by block number and log index desc.
func (storxscanPayments *storxscanPayments) List(ctx context.Context) (_ []storxscan.CachedPayment, err error) {
	defer mon.Task()(&ctx)(&err)

	dbxPmnts, err := storxscanPayments.db.All_StorxscanPayment_OrderBy_Asc_BlockNumber_Asc_LogIndex(ctx)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	var payments []storxscan.CachedPayment
	for _, dbxPmnt := range dbxPmnts {
		payments = append(payments, fromDBXPayment(dbxPmnt))
	}

	return payments, nil
}

// ListWallet returns list of storxscan payments order by block number and log index desc.
func (storxscanPayments *storxscanPayments) ListWallet(ctx context.Context, wallet blockchain.Address, limit int, offset int64) ([]storxscan.CachedPayment, error) {
	dbxPmnts, err := storxscanPayments.db.Limited_StorxscanPayment_By_ToAddress_OrderBy_Desc_BlockNumber_Desc_LogIndex(ctx,
		dbx.StorxscanPayment_ToAddress(wallet[:]),
		limit, offset)
	if err != nil {
		if errs.Is(err, sql.ErrNoRows) {
			return []storxscan.CachedPayment{}, nil
		}
		return nil, Error.Wrap(err)
	}

	var payments []storxscan.CachedPayment
	for _, dbxPmnt := range dbxPmnts {
		payments = append(payments, fromDBXPayment(dbxPmnt))
	}

	return payments, nil
}

// LastBlock returns the highest block known to DB.
func (storxscanPayments *storxscanPayments) LastBlock(ctx context.Context, status payments.PaymentStatus) (_ int64, err error) {
	defer mon.Task()(&ctx)(&err)

	blockNumber, err := storxscanPayments.db.First_StorxscanPayment_BlockNumber_By_Status_OrderBy_Desc_BlockNumber_Desc_LogIndex(
		ctx, dbx.StorxscanPayment_Status(string(status)))
	if err != nil {
		return 0, Error.Wrap(err)
	}
	if blockNumber == nil {
		return 0, Error.Wrap(storxscan.ErrNoPayments)
	}

	return blockNumber.BlockNumber, nil
}

// DeletePending removes all pending transactions from the DB.
func (storxscanPayments storxscanPayments) DeletePending(ctx context.Context) error {
	_, err := storxscanPayments.db.Delete_StorxscanPayment_By_Status(ctx,
		dbx.StorxscanPayment_Status(payments.PaymentStatusPending))
	return err
}

func (storxscanPayments storxscanPayments) ListConfirmed(ctx context.Context, blockNumber int64, logIndex int) (_ []storxscan.CachedPayment, err error) {
	defer mon.Task()(&ctx)(&err)

	query := `SELECT block_hash, block_number, transaction, log_index, from_address, to_address, token_value, usd_value, status, timestamp
              FROM storxscan_payments WHERE (storxscan_payments.block_number, storxscan_payments.log_index) > (?, ?) AND storxscan_payments.status = ?
              ORDER BY storxscan_payments.block_number, storxscan_payments.log_index`
	rows, err := storxscanPayments.db.Query(ctx, storxscanPayments.db.Rebind(query), blockNumber, logIndex, payments.PaymentStatusConfirmed)
	if err != nil {
		return nil, err
	}
	defer func() { err = errs.Combine(err, rows.Close()) }()

	var payments []storxscan.CachedPayment
	for rows.Next() {
		var payment dbx.StorxscanPayment
		err = rows.Scan(&payment.BlockHash, &payment.BlockNumber, &payment.Transaction, &payment.LogIndex,
			&payment.FromAddress, &payment.ToAddress, &payment.TokenValue, &payment.UsdValue, &payment.Status, &payment.Timestamp)
		if err != nil {
			return nil, err
		}
		payments = append(payments, fromDBXPayment(&payment))
	}
	return payments, rows.Err()
}

// fromDBXPayment converts dbx storxscan payment type to storxscan.CachedPayment.
func fromDBXPayment(dbxPmnt *dbx.StorxscanPayment) storxscan.CachedPayment {
	payment := storxscan.CachedPayment{
		TokenValue:  currency.AmountFromBaseUnits(dbxPmnt.TokenValue, currency.StorxToken),
		USDValue:    currency.AmountFromBaseUnits(dbxPmnt.UsdValue, currency.USDollarsMicro),
		Status:      payments.PaymentStatus(dbxPmnt.Status),
		BlockNumber: dbxPmnt.BlockNumber,
		LogIndex:    dbxPmnt.LogIndex,
		Timestamp:   dbxPmnt.Timestamp.UTC(),
	}
	copy(payment.From[:], dbxPmnt.FromAddress)
	copy(payment.To[:], dbxPmnt.ToAddress)
	copy(payment.BlockHash[:], dbxPmnt.BlockHash)
	copy(payment.Transaction[:], dbxPmnt.Transaction)
	return payment
}
