// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package trust_test

import (
	"common/testrand"
	"storx/storagenode/trust"
)

func makeSatelliteURL(host string) trust.SatelliteURL {
	return trust.SatelliteURL{
		ID:   testrand.NodeID(),
		Host: host,
		Port: 7777,
	}
}
