// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package stripecoinpayments

import (
	"context"
	"time"

	"github.com/stripe/stripe-go/v72"

	"common/uuid"
	"storx/satellite/payments"
)

// ensures that coupons implements payments.Coupons.
var _ payments.Coupons = (*coupons)(nil)

// coupons is an implementation of payments.Coupons.
//
// architecture: Service
type coupons struct {
	service *Service
}

// ApplyFreeTierCoupon applies the default free tier coupon to the account.
func (coupons *coupons) ApplyFreeTierCoupon(ctx context.Context, userID uuid.UUID) (_ *payments.Coupon, err error) {
	defer mon.Task()(&ctx)(&err)

	customerID, err := coupons.service.db.Customers().GetCustomerID(ctx, userID)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	customer, err := coupons.service.stripeClient.Customers().Update(customerID, &stripe.CustomerParams{
		Params: stripe.Params{Context: ctx},
		Coupon: stripe.String(coupons.service.StripeFreeTierCouponID),
	})
	if err != nil {
		return nil, Error.Wrap(err)
	}

	return coupons.service.stripeDiscountToPaymentsCoupon(customer.Discount)
}

// ApplyCoupon applies the coupon to account if it exists.
func (coupons *coupons) ApplyCoupon(ctx context.Context, userID uuid.UUID, couponID string) (_ *payments.Coupon, err error) {
	defer mon.Task()(&ctx)(&err)

	customerID, err := coupons.service.db.Customers().GetCustomerID(ctx, userID)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	customer, err := coupons.service.stripeClient.Customers().Update(customerID, &stripe.CustomerParams{
		Params: stripe.Params{Context: ctx},
		Coupon: stripe.String(couponID),
	})
	if err != nil {
		return nil, Error.Wrap(err)
	}
	return coupons.service.stripeDiscountToPaymentsCoupon(customer.Discount)
}

// ApplyCouponCode attempts to apply a coupon code to the user via Stripe.
func (coupons *coupons) ApplyCouponCode(ctx context.Context, userID uuid.UUID, couponCode string) (_ *payments.Coupon, err error) {
	defer mon.Task()(&ctx, userID, couponCode)(&err)

	user, err := coupons.service.usersDB.Get(ctx, userID)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	if user.UserAgent != nil {
		partner := string(user.UserAgent)
		if plan, ok := coupons.service.packagePlans[partner]; ok {
			coupon, err := coupons.GetByUserID(ctx, userID)
			if err != nil {
				return nil, err
			}
			if coupon != nil && coupon.ID == plan.CouponID {
				return nil, payments.ErrCouponConflict.New("coupon for partner '%s' should not be replaced", partner)
			}
		}
	}

	promoCodeIter := coupons.service.stripeClient.PromoCodes().List(&stripe.PromotionCodeListParams{
		ListParams: stripe.ListParams{Context: ctx},
		Code:       stripe.String(couponCode),
	})
	if !promoCodeIter.Next() {
		return nil, payments.ErrInvalidCoupon.New("Invalid coupon code")
	}
	promoCode := promoCodeIter.PromotionCode()

	customerID, err := coupons.service.db.Customers().GetCustomerID(ctx, userID)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	params := &stripe.CustomerParams{
		Params:        stripe.Params{Context: ctx},
		PromotionCode: stripe.String(promoCode.ID),
	}
	params.AddExpand("discount.promotion_code")

	customer, err := coupons.service.stripeClient.Customers().Update(customerID, params)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	if customer.Discount == nil || customer.Discount.Coupon == nil {
		return nil, Error.New("invalid discount after coupon code application; user ID:%s, customer ID:%s", userID, customerID)
	}

	return coupons.service.stripeDiscountToPaymentsCoupon(customer.Discount)
}

// GetByUserID returns the coupon applied to the user.
func (coupons *coupons) GetByUserID(ctx context.Context, userID uuid.UUID) (_ *payments.Coupon, err error) {
	defer mon.Task()(&ctx, userID)(&err)

	customerID, err := coupons.service.db.Customers().GetCustomerID(ctx, userID)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	params := &stripe.CustomerParams{Params: stripe.Params{Context: ctx}}
	params.AddExpand("discount.promotion_code")

	customer, err := coupons.service.stripeClient.Customers().Get(customerID, params)
	if err != nil {
		return nil, err
	}

	if customer.Discount == nil || customer.Discount.Coupon == nil {
		return nil, nil
	}

	return coupons.service.stripeDiscountToPaymentsCoupon(customer.Discount)
}

// stripeDiscountToPaymentsCoupon converts a Stripe discount to a payments.Coupon.
func (service *Service) stripeDiscountToPaymentsCoupon(dc *stripe.Discount) (coupon *payments.Coupon, err error) {
	if dc == nil {
		return nil, Error.New("discount is nil")
	}

	if dc.Coupon == nil {
		return nil, Error.New("discount.Coupon is nil")
	}

	var partnered bool
	for _, plan := range service.packagePlans {
		if plan.CouponID == dc.Coupon.ID {
			partnered = true
			break
		}
	}

	coupon = &payments.Coupon{
		ID:         dc.Coupon.ID,
		Name:       dc.Coupon.Name,
		AmountOff:  dc.Coupon.AmountOff,
		PercentOff: dc.Coupon.PercentOff,
		AddedAt:    time.Unix(dc.Start, 0),
		ExpiresAt:  time.Unix(dc.End, 0),
		Duration:   payments.CouponDuration(dc.Coupon.Duration),
		Partnered:  partnered,
	}

	if dc.PromotionCode != nil {
		coupon.PromoCode = dc.PromotionCode.Code
	}

	return coupon, nil
}
