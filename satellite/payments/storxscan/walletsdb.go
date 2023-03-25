// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

package storxscan

import (
	"context"

	"common/uuid"
	"storx/private/blockchain"
)

// WalletsDB is an interface which defines functionality
// of DB which stores user storxscan wallets.
//
// architecture: Database
type WalletsDB interface {
	// Add adds a new storxscan wallet to the DB and associates it with a user.
	Add(ctx context.Context, userID uuid.UUID, walletAddress blockchain.Address) error
	// GetWallet returns the wallet address associated with the given user.
	GetWallet(ctx context.Context, userID uuid.UUID) (blockchain.Address, error)
	// GetUser returns the userID associated with the given wallet.
	GetUser(ctx context.Context, wallet blockchain.Address) (uuid.UUID, error)
	// GetAll returns all saved wallet entries.
	GetAll(ctx context.Context) (_ []Wallet, err error)
}

// Wallet associates a user ID and a wallet address.
type Wallet struct {
	UserID  uuid.UUID
	Address blockchain.Address
}
