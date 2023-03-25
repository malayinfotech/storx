// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

import { ErrorConflict } from './errors/ErrorConflict';
import { ErrorTooManyRequests } from './errors/ErrorTooManyRequests';

import {
    AccountBalance,
    Coupon,
    CreditCard,
    PaymentsApi,
    PaymentsHistoryItem,
    ProjectUsageAndCharges,
    ProjectUsagePriceModel,
    TokenAmount,
    NativePaymentHistoryItem,
    Wallet,
} from '@/types/payments';
import { HttpClient } from '@/utils/httpClient';
import { Time } from '@/utils/time';

/**
 * PaymentsHttpApi is a http implementation of Payments API.
 * Exposes all payments-related functionality
 */
export class PaymentsHttpApi implements PaymentsApi {
    private readonly client: HttpClient = new HttpClient();
    private readonly ROOT_PATH: string = '/api/v0/payments';

    /**
     * Get account balance.
     *
     * @returns balance in cents
     * @throws Error
     */
    public async getBalance(): Promise<AccountBalance> {
        const path = `${this.ROOT_PATH}/account/balance`;
        const response = await this.client.get(path);

        if (!response.ok) {
            throw new Error('Can not get account balance');
        }

        const balance = await response.json();
        if (balance) {
            return new AccountBalance(balance.freeCredits, balance.coins, balance.credits);
        }

        return new AccountBalance();
    }

    /**
     * Try to set up a payment account.
     *
     * @throws Error
     */
    public async setupAccount(): Promise<string> {
        const path = `${this.ROOT_PATH}/account`;
        const response = await this.client.post(path, null);
        const couponType = await response.json();

        if (response.ok) {
            return couponType;
        }

        throw new Error('can not setup account');
    }

    /**
     * projectsUsageAndCharges returns usage and how much money current user will be charged for each project which he owns.
     */
    public async projectsUsageAndCharges(start: Date, end: Date): Promise<ProjectUsageAndCharges[]> {
        const since = Time.toUnixTimestamp(start).toString();
        const before = Time.toUnixTimestamp(end).toString();
        const path = `${this.ROOT_PATH}/account/charges?from=${since}&to=${before}`;
        const response = await this.client.get(path);

        if (!response.ok) {
            throw new Error('can not get projects charges');
        }

        const charges = await response.json();
        if (charges) {
            return charges.map(charge =>
                new ProjectUsageAndCharges(
                    new Date(charge.since),
                    new Date(charge.before),
                    charge.egress,
                    charge.storage,
                    charge.segmentCount,
                    charge.projectId,
                    charge.storagePrice,
                    charge.egressPrice,
                    charge.segmentPrice,
                ),
            );
        }

        return [];
    }

    /**
     * projectUsagePriceModel returns usage and how much money current user will be charged for each project which he owns.
     */
    public async projectUsagePriceModel(): Promise<ProjectUsagePriceModel> {
        const path = `${this.ROOT_PATH}/pricing`;
        const response = await this.client.get(path);

        if (!response.ok) {
            throw new Error('cannot get project usage price model');
        }

        const model = await response.json();
        if (model) {
            return new ProjectUsagePriceModel(model.storageMBMonthCents, model.egressMBCents, model.segmentMonthCents);
        }

        return new ProjectUsagePriceModel();
    }

    /**
     * Add credit card.
     *
     * @param token - stripe token used to add a credit card as a payment method
     * @throws Error
     */
    public async addCreditCard(token: string): Promise<void> {
        const path = `${this.ROOT_PATH}/cards`;
        const response = await this.client.post(path, token);

        if (response.ok) {
            return;
        }

        throw new Error('can not add credit card');
    }

    /**
     * Detach credit card from payment account.
     *
     * @param cardId
     * @throws Error
     */
    public async removeCreditCard(cardId: string): Promise<void> {
        const path = `${this.ROOT_PATH}/cards/${cardId}`;
        const response = await this.client.delete(path);

        if (response.ok) {
            return;
        }

        throw new Error('can not remove credit card');
    }

    /**
     * Get list of user`s credit cards.
     *
     * @returns list of credit cards
     * @throws Error
     */
    public async listCreditCards(): Promise<CreditCard[]> {
        const path = `${this.ROOT_PATH}/cards`;
        const response = await this.client.get(path);

        if (!response.ok) {
            throw new Error('can not list credit cards');
        }

        const creditCards = await response.json();

        if (creditCards) {
            return creditCards.map(card => new CreditCard(card.id, card.expMonth, card.expYear, card.brand, card.last4, card.isDefault));
        }

        return [];
    }

    /**
     * Make credit card default.
     *
     * @param cardId
     * @throws Error
     */
    public async makeCreditCardDefault(cardId: string): Promise<void> {
        const path = `${this.ROOT_PATH}/cards`;
        const response = await this.client.patch(path, cardId);

        if (response.ok) {
            return;
        }

        throw new Error('can not make credit card default');
    }

    /**
     * Returns a list of invoices, transactions and all others payments history items for payment account.
     *
     * @returns list of payments history items
     * @throws Error
     */
    public async paymentsHistory(): Promise<PaymentsHistoryItem[]> {
        const path = `${this.ROOT_PATH}/billing-history`;
        const response = await this.client.get(path);

        if (!response.ok) {
            throw new Error('can not list billing history');
        }

        const paymentsHistoryItems = await response.json();
        if (paymentsHistoryItems) {
            return paymentsHistoryItems.map(item =>
                new PaymentsHistoryItem(
                    item.id,
                    item.description,
                    item.amount,
                    item.received,
                    item.status,
                    item.link,
                    new Date(item.start),
                    new Date(item.end),
                    item.type,
                    item.remaining,
                ),
            );
        }

        return [];
    }

    /**
     * Returns a list of native token payments.
     *
     * @returns list of native token payment history items
     * @throws Error
     */
    public async nativePaymentsHistory(): Promise<NativePaymentHistoryItem[]> {
        const path = `${this.ROOT_PATH}/wallet/payments`;
        const response = await this.client.get(path);

        if (!response.ok) {
            throw new Error('Can not list token payment history');
        }

        const json = await response.json();
        if (!json) return  [];
        if (json.payments) {
            return json.payments.map(item =>
                new NativePaymentHistoryItem(
                    item.ID,
                    item.Wallet,
                    item.Type,
                    new TokenAmount(item.Amount.value, item.Amount.currency),
                    new TokenAmount(item.Received.value, item.Received.currency),
                    item.Status,
                    item.Link,
                    new Date(item.Timestamp),
                ),
            );
        }

        return [];
    }

    /**
     * applyCouponCode applies a coupon code.
     *
     * @param couponCode
     * @throws Error
     */
    public async applyCouponCode(couponCode: string): Promise<Coupon> {
        const path = `${this.ROOT_PATH}/coupon/apply`;
        const response = await this.client.patch(path, couponCode);
        const errMsg = `Could not apply coupon code "${couponCode}"`;

        if (!response.ok) {
            switch (response.status) {
            case 409:
                throw new ErrorConflict('You currently have an active coupon. Please try again when your coupon is no longer active, or contact Support for further help.');
            case 429:
                throw new ErrorTooManyRequests('You\'ve exceeded limit of attempts, try again in 5 minutes');
            default:
                throw new Error(errMsg);
            }
        }

        const coupon = await response.json();

        if (!coupon) {
            throw new Error(errMsg);
        }

        return new Coupon(
            coupon.id,
            coupon.promoCode,
            coupon.name,
            coupon.amountOff,
            coupon.percentOff,
            new Date(coupon.addedAt),
            coupon.expiresAt ? new Date(coupon.expiresAt) : null,
            coupon.duration,
            coupon.partnered,
        );
    }

    /**
     * getCoupon returns the coupon applied to the user.
     *
     * @throws Error
     */
    public async getCoupon(): Promise<Coupon | null> {
        const path = `${this.ROOT_PATH}/coupon`;
        const response = await this.client.get(path);
        if (!response.ok) {
            throw new Error('cannot retrieve coupon');
        }

        const coupon = await response.json();

        if (!coupon) {
            return null;
        }

        return new Coupon(
            coupon.id,
            coupon.promoCode,
            coupon.name,
            coupon.amountOff,
            coupon.percentOff,
            new Date(coupon.addedAt),
            coupon.expiresAt ? new Date(coupon.expiresAt) : null,
            coupon.duration,
            coupon.partnered,
        );
    }

    /**
     * Get native storx token wallet.
     *
     * @returns wallet
     * @throws Error
     */
    public async getWallet(): Promise<Wallet> {
        const path = `${this.ROOT_PATH}/wallet`;
        const response = await this.client.get(path);

        if (!response.ok) {
            switch (response.status) {
            case 404:
                return new Wallet();
            default:
                throw new Error('Can not get wallet');
            }
        }

        const wallet = await response.json();
        if (wallet) {
            return new Wallet(wallet.address, wallet.balance);
        }

        throw new Error('Can not get wallet');
    }

    /**
     * Claim new native storx token wallet.
     *
     * @returns wallet
     * @throws Error
     */
    public async claimWallet(): Promise<Wallet> {
        const path = `${this.ROOT_PATH}/wallet`;
        const response = await this.client.post(path, null);

        if (!response.ok) {
            throw new Error('Can not claim new wallet');
        }

        const wallet = await response.json();
        if (wallet) {
            return new Wallet(wallet.address, wallet.balance);
        }

        return new Wallet();
    }

    /**
     * Purchases the pricing package associated with the user's partner.
     *
     * @param token - the Stripe token used to add a credit card as a payment method
     * @throws Error
     */
    public async purchasePricingPackage(token: string): Promise<void> {
        const path = `${this.ROOT_PATH}/purchase-package`;
        const response = await this.client.post(path, token);

        if (response.ok) {
            return;
        }

        throw new Error('Could not purchase pricing package');
    }

    /**
     * Returns whether there is a pricing package configured for the user's partner.
     *
     * @throws Error
     */
    public async pricingPackageAvailable(): Promise<boolean> {
        const path = `${this.ROOT_PATH}/package-available`;
        const response = await this.client.get(path);

        if (response.ok) {
            return await response.json();
        }

        throw new Error('Could not check pricing package availability');
    }
}
