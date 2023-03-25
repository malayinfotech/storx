// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

package metabasetest

import (
	"time"

	"common/storx"
	"storx/satellite/metabase"
)

// DefaultRedundancy contains default redundancy scheme.
var DefaultRedundancy = storx.RedundancyScheme{
	Algorithm:      storx.ReedSolomon,
	ShareSize:      2048,
	RequiredShares: 1,
	RepairShares:   1,
	OptimalShares:  1,
	TotalShares:    1,
}

// DefaultEncryption contains default encryption parameters.
var DefaultEncryption = storx.EncryptionParameters{
	CipherSuite: storx.EncAESGCM,
	BlockSize:   29 * 256,
}

// DefaultRawSegment returns default raw segment.
func DefaultRawSegment(obj metabase.ObjectStream, segmentPosition metabase.SegmentPosition) metabase.RawSegment {
	return metabase.RawSegment{
		StreamID:    obj.StreamID,
		Position:    segmentPosition,
		RootPieceID: storx.PieceID{1},
		Pieces:      metabase.Pieces{{Number: 0, StorageNode: storx.NodeID{2}}},
		CreatedAt:   time.Now(),

		EncryptedKey:      []byte{3},
		EncryptedKeyNonce: []byte{4},
		EncryptedETag:     []byte{5},

		EncryptedSize: 1024,
		PlainSize:     512,
		PlainOffset:   0,
		Redundancy:    DefaultRedundancy,
	}
}
