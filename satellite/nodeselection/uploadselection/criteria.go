// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information

package uploadselection

import (
	"common/storx"
	"common/storx/location"
)

// Criteria to filter nodes.
type Criteria struct {
	ExcludeNodeIDs       []storx.NodeID
	AutoExcludeSubnets   map[string]struct{} // initialize it with empty map to keep only one node per subnet.
	Placement            storx.PlacementConstraint
	ExcludedCountryCodes []location.CountryCode
}

// MatchInclude returns with true if node is selected.
func (c *Criteria) MatchInclude(node *Node) bool {
	if ContainsID(c.ExcludeNodeIDs, node.ID) {
		return false
	}

	if !c.Placement.AllowedCountry(node.CountryCode) {
		return false
	}

	if c.AutoExcludeSubnets != nil {
		if _, excluded := c.AutoExcludeSubnets[node.LastNet]; excluded {
			return false
		}
		c.AutoExcludeSubnets[node.LastNet] = struct{}{}
	}

	for _, code := range c.ExcludedCountryCodes {
		if code.String() == "" {
			continue
		}
		if node.CountryCode == code {
			return false
		}
	}

	return true
}

// ContainsID returns whether ids contain id.
func ContainsID(ids []storx.NodeID, id storx.NodeID) bool {
	for _, k := range ids {
		if k == id {
			return true
		}
	}
	return false
}
