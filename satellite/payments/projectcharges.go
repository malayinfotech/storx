// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package payments

import (
	"github.com/shopspring/decimal"

	"common/uuid"
	"storx/satellite/accounting"
)

// ProjectCharge shows project usage and how much money current project will charge in the end of the month.
type ProjectCharge struct {
	accounting.ProjectUsage

	ProjectID uuid.UUID `json:"projectId"`
	// StorageGbHrs shows how much cents we should pay for storing GB*Hrs.
	StorageGbHrs int64 `json:"storagePrice"`
	// Egress shows how many cents we should pay for Egress.
	Egress int64 `json:"egressPrice"`
	// SegmentCount shows how many cents we should pay for objects count.
	SegmentCount int64 `json:"segmentPrice"`
}

// ProjectUsagePriceModel represents price model for project usage.
type ProjectUsagePriceModel struct {
	StorageMBMonthCents decimal.Decimal `json:"storageMBMonthCents"`
	EgressMBCents       decimal.Decimal `json:"egressMBCents"`
	SegmentMonthCents   decimal.Decimal `json:"segmentMonthCents"`
}
