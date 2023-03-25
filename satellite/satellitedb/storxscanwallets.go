// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package satellitedb

import (
	"context"
	"database/sql"
	"errors"

	"common/uuid"
	"storx/private/blockchain"
	"storx/satellite/payments/billing"
	"storx/satellite/payments/storxscan"
	"storx/satellite/satellitedb/dbx"
)

// ensure that storxscanWalletsDB implements storxscan.WalletsDB.
var _ storxscan.WalletsDB = (*storxscanWalletsDB)(nil)

// storxscanWalletsDB is Storxscan wallets DB.
//
// architecture: Database
type storxscanWalletsDB struct {
	db *satelliteDB
}

// Add creates new user/wallet association record.
func (walletsDB storxscanWalletsDB) Add(ctx context.Context, userID uuid.UUID, walletAddress blockchain.Address) (err error) {
	defer mon.Task()(&ctx)(&err)
	return walletsDB.db.CreateNoReturn_StorxscanWallet(ctx,
		dbx.StorxscanWallet_UserId(userID[:]),
		dbx.StorxscanWallet_WalletAddress(walletAddress.Bytes()))
}

// GetWallet returns the wallet associated with the given user.
func (walletsDB storxscanWalletsDB) GetWallet(ctx context.Context, userID uuid.UUID) (_ blockchain.Address, err error) {
	defer mon.Task()(&ctx)(&err)
	wallet, err := walletsDB.db.Get_StorxscanWallet_WalletAddress_By_UserId(ctx, dbx.StorxscanWallet_UserId(userID[:]))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return blockchain.Address{}, billing.ErrNoWallet
		}
		return blockchain.Address{}, Error.Wrap(err)
	}
	address, err := blockchain.BytesToAddress(wallet.WalletAddress)
	if err != nil {
		return blockchain.Address{}, Error.Wrap(err)
	}
	return address, nil
}

// GetUser returns the userID associated with the given wallet.
func (walletsDB storxscanWalletsDB) GetUser(ctx context.Context, walletAddress blockchain.Address) (_ uuid.UUID, err error) {
	defer mon.Task()(&ctx)(&err)
	userID, err := walletsDB.db.Get_StorxscanWallet_UserId_By_WalletAddress(ctx, dbx.StorxscanWallet_WalletAddress(walletAddress.Bytes()))
	if err != nil {
		return uuid.UUID{}, Error.Wrap(err)
	}
	id, err := uuid.FromBytes(userID.UserId)
	if err != nil {
		return uuid.UUID{}, Error.Wrap(err)
	}
	return id, nil
}

// GetAll returns all saved wallet entries.
func (walletsDB storxscanWalletsDB) GetAll(ctx context.Context) (_ []storxscan.Wallet, err error) {
	defer mon.Task()(&ctx)(&err)
	entries, err := walletsDB.db.All_StorxscanWallet(ctx)
	if err != nil {
		return nil, Error.Wrap(err)
	}
	var wallets []storxscan.Wallet
	for _, entry := range entries {
		userID, err := uuid.FromBytes(entry.UserId)
		if err != nil {
			return nil, Error.Wrap(err)
		}
		address, err := blockchain.BytesToAddress(entry.WalletAddress)
		if err != nil {
			return nil, Error.Wrap(err)
		}
		wallets = append(wallets, storxscan.Wallet{UserID: userID, Address: address})
	}
	return wallets, nil
}
