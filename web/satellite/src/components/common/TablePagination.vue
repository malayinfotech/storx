// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

<template>
    <div class="pagination-container">
        <div class="pagination-container__pages">
            <span class="pagination-container__pages__label">{{ label }}</span>
            <div class="pagination-container__button" @click="prevPage">
                <PaginationRightIcon class="pagination-container__button__image reversed" />
            </div>
            <div class="pagination-container__button" @click="nextPage">
                <PaginationRightIcon class="pagination-container__button__image" />
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';

import { OnPageClickCallback } from '@/types/pagination';
import { useLoading } from '@/composables/useLoading';

import PaginationRightIcon from '@/../static/images/common/tablePaginationArrowRight.svg';

const { withLoading } = useLoading();

const props = withDefaults(defineProps<{
    totalPageCount?: number;
    limit?: number;
    totalItemsCount?: number;
    onPageClickCallback?: OnPageClickCallback;
}>(), {
    totalPageCount: 0,
    limit: 0,
    totalItemsCount: 0,
    onPageClickCallback: () => () => new Promise(() => false),
});

const currentPageNumber = ref<number>(1);

const label = computed((): string => {
    const currentMaxPage = currentPageNumber.value * props.limit > props.totalItemsCount ?
        props.totalItemsCount
        : currentPageNumber.value * props.limit;
    return `${currentPageNumber.value * props.limit - props.limit + 1} - ${currentMaxPage} of ${props.totalItemsCount}`;
});

const isFirstPage = computed((): boolean => {
    return currentPageNumber.value === 1;
});

const isLastPage = computed((): boolean => {
    return currentPageNumber.value === props.totalPageCount;
});

/**
 * nextPage fires after 'next' arrow click.
 */
async function nextPage(): Promise<void> {
    await withLoading(async () => {
        if (isLastPage.value) {
            return;
        }

        await props.onPageClickCallback(currentPageNumber.value + 1);
        incrementCurrentPage();
    });
}

/**
 * prevPage fires after 'previous' arrow click.
 */
async function prevPage(): Promise<void> {
    await withLoading(async () => {
        if (isFirstPage.value) {
            return;
        }

        await props.onPageClickCallback(currentPageNumber.value - 1);
        decrementCurrentPage();
    });
}

function incrementCurrentPage(): void {
    currentPageNumber.value++;
}

function decrementCurrentPage(): void {
    currentPageNumber.value--;
}
</script>

<style scoped lang="scss">
.pagination-container {
    display: flex;
    align-items: center;
    justify-content: flex-end;

    &__pages {
        display: flex;
        align-items: center;

        &__label {
            margin-right: 25px;
            font-family: 'font_regular', sans-serif;
            font-size: 14px;
            line-height: 24px;
            color: rgb(44 53 58 / 60%);
        }
    }

    &__button {
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        width: 15px;
        height: 15px;
        max-width: 15px;
        max-height: 15px;
    }
}

.reversed {
    transform: rotate(180deg);
}
</style>
