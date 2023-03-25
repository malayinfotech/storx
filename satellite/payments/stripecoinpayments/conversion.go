// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package stripecoinpayments

import (
	"github.com/shopspring/decimal"

	"common/currency"
)

// convertToCents convert amount to USD cents with given rate.
func convertToCents(rate decimal.Decimal, amount currency.Amount) int64 {
	amountDecimal := amount.AsDecimal()
	usd := amountDecimal.Mul(rate)
	usdCents := usd.Shift(2)
	return usdCents.Round(0).IntPart()
}
