// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package pricing

import (
	"context"

	"common/storx"
)

// DB works with pricing database.
//
// architecture: Database
type DB interface {
	// Store inserts or updates pricing model into the DB.
	Store(ctx context.Context, stats Pricing) error
	// Get retrieves pricing model for specific satellite.
	Get(ctx context.Context, satelliteID storx.NodeID) (*Pricing, error)
}

// Pricing consist pricing model for storagenode.
type Pricing struct {
	SatelliteID     storx.NodeID `json:"satelliteID"`
	EgressBandwidth int64        `json:"egressBandwidth"`
	RepairBandwidth int64        `json:"repairBandwidth"`
	AuditBandwidth  int64        `json:"auditBandwidth"`
	DiskSpace       int64        `json:"diskSpace"`
}
