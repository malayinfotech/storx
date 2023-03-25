// Copyright (C) 2018 Storx Labs, Inc.
// See LICENSE for copying information.

package overlay

import (
	"context"

	"common/identity"
	"common/storx"
)

// PeerIdentities stores storagenode peer identities.
//
// architecture: Database
type PeerIdentities interface {
	// Set adds a peer identity entry for a node
	Set(context.Context, storx.NodeID, *identity.PeerIdentity) error
	// Get gets peer identity
	Get(context.Context, storx.NodeID) (*identity.PeerIdentity, error)
	// BatchGet gets all nodes peer identities in a transaction
	BatchGet(context.Context, storx.NodeIDList) ([]*identity.PeerIdentity, error)
}
