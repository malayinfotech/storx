// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

import {
    AccountBalance,
    Coupon,
    CreditCard,
    PaymentsApi,
    PaymentsHistoryItem,
    ProjectUsageAndCharges,
    ProjectUsagePriceModel,
    TokenDeposit,
    NativePaymentHistoryItem,
    Wallet,
} from '@/types/payments';

/**
 * Mock for PaymentsApi
 */
export class PaymentsMock implements PaymentsApi {
    private mockCoupon: Coupon | null = null;

    public setMockCoupon(coupon: Coupon | null): void {
        this.mockCoupon = coupon;
    }

    setupAccount(): Promise<string> {
        throw new Error('Method not implemented');
    }

    getBalance(): Promise<AccountBalance> {
        return Promise.resolve(new AccountBalance());
    }

    projectsUsageAndCharges(): Promise<ProjectUsageAndCharges[]> {
        return Promise.resolve([]);
    }

    projectUsagePriceModel(): Promise<ProjectUsagePriceModel> {
        return Promise.resolve(new ProjectUsagePriceModel('1', '1', '1'));
    }

    addCreditCard(_token: string): Promise<void> {
        throw new Error('Method not implemented');
    }

    removeCreditCard(_cardId: string): Promise<void> {
        throw new Error('Method not implemented');
    }

    listCreditCards(): Promise<CreditCard[]> {
        return Promise.resolve([]);
    }

    makeCreditCardDefault(_cardId: string): Promise<void> {
        throw new Error('Method not implemented');
    }

    paymentsHistory(): Promise<PaymentsHistoryItem[]> {
        return Promise.resolve([]);
    }

    nativePaymentsHistory(): Promise<NativePaymentHistoryItem[]> {
        return Promise.resolve([]);
    }

    makeTokenDeposit(amount: number): Promise<TokenDeposit> {
        return Promise.resolve(new TokenDeposit(amount, 'testAddress', 'testLink'));
    }

    applyCouponCode(_: string): Promise<Coupon> {
        throw new Error('Method not implemented');
    }

    getCoupon(): Promise<Coupon | null> {
        return Promise.resolve(this.mockCoupon);
    }

    getWallet(): Promise<Wallet> {
        return Promise.resolve(new Wallet());
    }

    claimWallet(): Promise<Wallet> {
        return Promise.resolve(new Wallet());
    }

    purchasePricingPackage(_: string): Promise<void> {
        throw new Error('Method not implemented');
    }

    pricingPackageAvailable(): Promise<boolean> {
        throw new Error('Method not implemented');
    }
}
