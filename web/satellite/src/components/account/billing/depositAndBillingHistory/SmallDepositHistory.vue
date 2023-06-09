// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

<template>
    <div v-if="depositHistoryItems.length > 0" class="deposit-area">
        <div class="deposit-area__header">
            <h1 class="deposit-area__header__title">Short Balance History</h1>
            <div class="deposit-area__header__button" @click.stop="onViewAllClick">View All</div>
        </div>
        <SortingHeader />
        <PaymentsItem
            v-for="item in depositHistoryItems"
            :key="item.id"
            :billing-item="item"
        />
    </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import { RouteConfig } from '@/router';
import { PaymentsHistoryItem, PaymentsHistoryItemType } from '@/types/payments';
import { AnalyticsHttpApi } from '@/api/analytics';
import { useRouter, useStore } from '@/utils/hooks';

import SortingHeader from '@/components/account/billing/depositAndBillingHistory/SortingHeader.vue';
import PaymentsItem from '@/components/account/billing/depositAndBillingHistory/PaymentsItem.vue';

const analytics: AnalyticsHttpApi = new AnalyticsHttpApi();

const store = useStore();
const router = useRouter();

/**
 * Returns first 3 of deposit history items.
 */
const depositHistoryItems = computed((): PaymentsHistoryItem[] => {
    return store.state.paymentsModule.paymentsHistory.filter((item: PaymentsHistoryItem) => {
        return item.type === PaymentsHistoryItemType.Transaction || item.type === PaymentsHistoryItemType.DepositBonus;
    }).slice(0, 3);
});

/**
 * Changes location to deposit history route.
 */
function onViewAllClick(): void {
    analytics.pageVisit(RouteConfig.Account.with(RouteConfig.DepositHistory).path);
    router.push(RouteConfig.Account.with(RouteConfig.DepositHistory).path);
}
</script>

<style scoped lang="scss">
    h1,
    span {
        margin: 0;
        color: #354049;
    }

    .deposit-area {
        padding: 40px 40px 10px;
        background-color: #fff;
        border-radius: 8px;
        font-family: 'font_regular', sans-serif;

        &__header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 40px;
            font-family: 'font_bold', sans-serif;

            &__title {
                font-size: 28px;
                line-height: 42px;
            }

            &__button {
                display: flex;
                width: 120px;
                height: 48px;
                border: 1px solid #afb7c1;
                border-radius: 8px;
                align-items: center;
                justify-content: center;
                font-size: 16px;
                color: #354049;
                cursor: pointer;

                &:hover {
                    background-color: #2683ff;
                    color: #fff;
                }
            }
        }
    }
</style>
