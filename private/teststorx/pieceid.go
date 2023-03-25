// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package teststorx

import (
	"common/storx"
)

// PieceIDFromBytes converts a byte slice into a piece ID.
func PieceIDFromBytes(b []byte) storx.PieceID {
	id, _ := storx.PieceIDFromBytes(fit(b))
	return id
}

// PieceIDFromString decodes a hex encoded piece ID string.
func PieceIDFromString(s string) storx.PieceID {
	return PieceIDFromBytes([]byte(s))
}
