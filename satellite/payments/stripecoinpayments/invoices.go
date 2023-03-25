// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package stripecoinpayments

import (
	"context"
	"time"

	"github.com/stripe/stripe-go/v72"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"common/uuid"
	"storx/satellite/payments"
)

// invoices is an implementation of payments.Invoices.
//
// architecture: Service
type invoices struct {
	service *Service
}

func (invoices *invoices) Create(ctx context.Context, userID uuid.UUID, price int64, desc string) (*payments.Invoice, error) {
	customerID, err := invoices.service.db.Customers().GetCustomerID(ctx, userID)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	inv, err := invoices.service.stripeClient.Invoices().New(&stripe.InvoiceParams{
		Params:                      stripe.Params{Context: ctx},
		Customer:                    stripe.String(customerID),
		Discounts:                   []*stripe.InvoiceDiscountParams{},
		Description:                 stripe.String(desc),
		PendingInvoiceItemsBehavior: stripe.String("exclude"),
	})
	if err != nil {
		return nil, Error.Wrap(err)
	}

	item, err := invoices.service.stripeClient.InvoiceItems().New(&stripe.InvoiceItemParams{
		Params:      stripe.Params{Context: ctx},
		Customer:    stripe.String(customerID),
		Amount:      stripe.Int64(price),
		Description: stripe.String(desc),
		Currency:    stripe.String(string(stripe.CurrencyUSD)),
		Invoice:     stripe.String(inv.ID),
	})
	if err != nil {
		return nil, Error.Wrap(err)
	}

	return &payments.Invoice{
		ID:          inv.ID,
		Description: inv.Description,
		Amount:      item.Amount,
		Status:      string(inv.Status),
	}, nil
}

func (invoices *invoices) Pay(ctx context.Context, invoiceID, paymentMethodID string) (*payments.Invoice, error) {
	inv, err := invoices.service.stripeClient.Invoices().Pay(invoiceID, &stripe.InvoicePayParams{
		Params:        stripe.Params{Context: ctx},
		PaymentMethod: stripe.String(paymentMethodID),
	})
	if err != nil {
		return nil, Error.Wrap(err)
	}
	return &payments.Invoice{
		ID:          inv.ID,
		Description: inv.Description,
		Amount:      inv.AmountPaid,
		Status:      string(inv.Status),
	}, nil
}

// AttemptPayOverdueInvoices attempts to pay a user's open, overdue invoices.
func (invoices *invoices) AttemptPayOverdueInvoices(ctx context.Context, userID uuid.UUID) (err error) {
	customerID, err := invoices.service.db.Customers().GetCustomerID(ctx, userID)
	if err != nil {
		return Error.Wrap(err)
	}

	params := &stripe.InvoiceListParams{
		ListParams:   stripe.ListParams{Context: ctx},
		Customer:     &customerID,
		Status:       stripe.String(string(stripe.InvoiceStatusOpen)),
		DueDateRange: &stripe.RangeQueryParams{LesserThan: time.Now().Unix()},
	}

	var errGrp errs.Group

	invoicesIterator := invoices.service.stripeClient.Invoices().List(params)
	for invoicesIterator.Next() {
		stripeInvoice := invoicesIterator.Invoice()

		params := &stripe.InvoicePayParams{Params: stripe.Params{Context: ctx}}
		invResponse, err := invoices.service.stripeClient.Invoices().Pay(stripeInvoice.ID, params)
		if err != nil {
			errGrp.Add(Error.New("unable to pay invoice %s: %w", stripeInvoice.ID, err))
			continue
		}

		if invResponse != nil && invResponse.Status != stripe.InvoiceStatusPaid {
			errGrp.Add(Error.New("invoice not paid after payment triggered %s", stripeInvoice.ID))
		}

	}

	if err = invoicesIterator.Err(); err != nil {
		return Error.Wrap(err)
	}

	return errGrp.Err()
}

// List returns a list of invoices for a given payment account.
func (invoices *invoices) List(ctx context.Context, userID uuid.UUID) (invoicesList []payments.Invoice, err error) {
	defer mon.Task()(&ctx, userID)(&err)

	customerID, err := invoices.service.db.Customers().GetCustomerID(ctx, userID)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	params := &stripe.InvoiceListParams{
		ListParams: stripe.ListParams{Context: ctx},
		Customer:   &customerID,
	}

	invoicesIterator := invoices.service.stripeClient.Invoices().List(params)
	for invoicesIterator.Next() {
		stripeInvoice := invoicesIterator.Invoice()

		total := stripeInvoice.Total
		if stripeInvoice.Lines != nil {
			for _, line := range stripeInvoice.Lines.Data {
				// If amount is negative, this is a coupon or a credit line item.
				// Add them to the total.
				if line.Amount < 0 {
					total -= line.Amount
				}
			}
		}

		invoicesList = append(invoicesList, payments.Invoice{
			ID:          stripeInvoice.ID,
			CustomerID:  customerID,
			Description: stripeInvoice.Description,
			Amount:      total,
			Status:      convertStatus(stripeInvoice.Status),
			Link:        stripeInvoice.InvoicePDF,
			Start:       time.Unix(stripeInvoice.PeriodStart, 0),
		})
	}

	if err = invoicesIterator.Err(); err != nil {
		return nil, Error.Wrap(err)
	}

	return invoicesList, nil
}

func (invoices *invoices) ListFailed(ctx context.Context) (invoicesList []payments.Invoice, err error) {
	defer mon.Task()(&ctx)(&err)

	status := string(stripe.InvoiceStatusOpen)
	params := &stripe.InvoiceListParams{
		ListParams: stripe.ListParams{Context: ctx},
		Status:     &status,
	}

	invoicesIterator := invoices.service.stripeClient.Invoices().List(params)
	for invoicesIterator.Next() {
		stripeInvoice := invoicesIterator.Invoice()

		total := stripeInvoice.Total
		for _, line := range stripeInvoice.Lines.Data {
			// If amount is negative, this is a coupon or a credit line item.
			// Add them to the total.
			if line.Amount < 0 {
				total -= line.Amount
			}
		}

		if invoices.isInvoiceFailed(stripeInvoice) {
			invoicesList = append(invoicesList, payments.Invoice{
				ID:          stripeInvoice.ID,
				CustomerID:  stripeInvoice.Customer.ID,
				Description: stripeInvoice.Description,
				Amount:      total,
				Status:      string(stripeInvoice.Status),
				Link:        stripeInvoice.InvoicePDF,
				Start:       time.Unix(stripeInvoice.PeriodStart, 0),
			})
		}
	}

	if err = invoicesIterator.Err(); err != nil {
		return nil, Error.Wrap(err)
	}

	return invoicesList, nil
}

// ListWithDiscounts returns a list of invoices and coupon usages for a given payment account.
func (invoices *invoices) ListWithDiscounts(ctx context.Context, userID uuid.UUID) (invoicesList []payments.Invoice, couponUsages []payments.CouponUsage, err error) {
	defer mon.Task()(&ctx, userID)(&err)

	customerID, err := invoices.service.db.Customers().GetCustomerID(ctx, userID)
	if err != nil {
		return nil, nil, Error.Wrap(err)
	}

	params := &stripe.InvoiceListParams{
		ListParams: stripe.ListParams{Context: ctx},
		Customer:   &customerID,
	}
	params.AddExpand("data.total_discount_amounts.discount")

	invoicesIterator := invoices.service.stripeClient.Invoices().List(params)
	for invoicesIterator.Next() {
		stripeInvoice := invoicesIterator.Invoice()

		total := stripeInvoice.Total
		for _, line := range stripeInvoice.Lines.Data {
			// If amount is negative, this is a coupon or a credit line item.
			// Add them to the total.
			if line.Amount < 0 {
				total -= line.Amount
			}
		}

		invoicesList = append(invoicesList, payments.Invoice{
			ID:          stripeInvoice.ID,
			CustomerID:  customerID,
			Description: stripeInvoice.Description,
			Amount:      total,
			Status:      convertStatus(stripeInvoice.Status),
			Link:        stripeInvoice.InvoicePDF,
			Start:       time.Unix(stripeInvoice.PeriodStart, 0),
		})

		for _, dcAmt := range stripeInvoice.TotalDiscountAmounts {
			if dcAmt == nil {
				return nil, nil, Error.New("discount amount is nil")
			}

			dc := dcAmt.Discount

			coupon, err := invoices.service.stripeDiscountToPaymentsCoupon(dc)
			if err != nil {
				return nil, nil, Error.Wrap(err)
			}

			usage := payments.CouponUsage{
				Coupon:      *coupon,
				Amount:      dcAmt.Amount,
				PeriodStart: time.Unix(stripeInvoice.PeriodStart, 0),
				PeriodEnd:   time.Unix(stripeInvoice.PeriodEnd, 0),
			}

			if dc.PromotionCode != nil {
				usage.Coupon.PromoCode = dc.PromotionCode.Code
			}

			couponUsages = append(couponUsages, usage)
		}
	}

	if err = invoicesIterator.Err(); err != nil {
		return nil, nil, Error.Wrap(err)
	}

	return invoicesList, couponUsages, nil
}

// CheckPendingItems returns if pending invoice items for a given payment account exist.
func (invoices *invoices) CheckPendingItems(ctx context.Context, userID uuid.UUID) (existingItems bool, err error) {
	defer mon.Task()(&ctx, userID)(&err)

	customerID, err := invoices.service.db.Customers().GetCustomerID(ctx, userID)
	if err != nil {
		return false, Error.Wrap(err)
	}

	params := &stripe.InvoiceItemListParams{
		ListParams: stripe.ListParams{Context: ctx},
		Customer:   &customerID,
		Pending:    stripe.Bool(true),
	}

	itemIterator := invoices.service.stripeClient.InvoiceItems().List(params)
	for itemIterator.Next() {
		item := itemIterator.InvoiceItem()
		if item != nil {
			return true, nil
		}
	}

	if err = itemIterator.Err(); err != nil {
		return false, Error.Wrap(err)
	}

	return false, nil
}

// Delete a draft invoice.
func (invoices *invoices) Delete(ctx context.Context, id string) (_ *payments.Invoice, err error) {
	defer mon.Task()(&ctx)(&err)

	params := &stripe.InvoiceParams{Params: stripe.Params{Context: ctx}}
	stripeInvoice, err := invoices.service.stripeClient.Invoices().Del(id, params)
	if err != nil {
		return nil, Error.Wrap(err)
	}
	return &payments.Invoice{
		ID:          stripeInvoice.ID,
		Description: stripeInvoice.Description,
		Amount:      stripeInvoice.AmountDue,
		Status:      convertStatus(stripeInvoice.Status),
		Link:        stripeInvoice.InvoicePDF,
		Start:       time.Unix(stripeInvoice.PeriodStart, 0),
	}, nil
}

func convertStatus(stripestatus stripe.InvoiceStatus) string {
	var status string
	switch stripestatus {
	case stripe.InvoiceStatusDraft:
		status = payments.InvoiceStatusDraft
	case stripe.InvoiceStatusOpen:
		status = payments.InvoiceStatusOpen
	case stripe.InvoiceStatusPaid:
		status = payments.InvoiceStatusPaid
	case stripe.InvoiceStatusUncollectible:
		status = payments.InvoiceStatusUncollectible
	case stripe.InvoiceStatusVoid:
		status = payments.InvoiceStatusVoid
	default:
		status = string(stripestatus)
	}
	return status
}

// isInvoiceFailed returns whether an invoice has failed.
func (invoices *invoices) isInvoiceFailed(invoice *stripe.Invoice) bool {
	if invoice.DueDate > 0 {
		// https://github.com/storx/storx/blob/77bf88e916a10dc898ebb594eafac667ed4426cd/satellite/payments/stripecoinpayments/service.go#L781-L787
		invoices.service.log.Info("Skipping invoice marked for manual payment",
			zap.String("id", invoice.ID),
			zap.String("number", invoice.Number),
			zap.String("customer", invoice.Customer.ID))
		return false
	}
	// https://stripe.com/docs/api/invoices/retrieve
	if invoice.NextPaymentAttempt > 0 {
		// stripe will automatically retry collecting payment.
		return false
	}

	return true
}
