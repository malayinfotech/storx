// Copyright (C) 2023 Storx Labs, Inc.
// See LICENSE for copying information.

<template>
    <table-item
        :on-click="onBack"
        :on-primary-click="onBack"
        :item="{'name': 'Back', 'size': '', 'date': ''}"
        item-type="back"
    >
        <th slot="options" />
    </table-item>
</template>

<script setup lang="ts">
import TableItem from '@/components/common/TableItem.vue';

const props = defineProps<{
    onBack: () => void;
}>();

</script>
