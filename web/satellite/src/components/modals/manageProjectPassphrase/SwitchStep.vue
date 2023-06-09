// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

<template>
    <div class="switch-step">
        <h1 class="switch-step__title">Switch passphrase</h1>
        <p class="switch-step__info">
            Switch passphrases to view existing data that is uploaded with a different passphrase, or upload new data.
            Please note that you won’t see the previous data once you switch passphrases.
        </p>
        <VInput
            label="Encryption Passphrase"
            :is-password="true"
            width="100%"
            height="56px"
            placeholder="Enter Encryption Passphrase"
            :error="enterError"
            @setData="setPassphrase"
        />
        <div class="switch-step__buttons">
            <VButton
                label="Back"
                width="100%"
                height="48px"
                :is-white="true"
                :on-press="onCancel"
            />
            <VButton
                label="Switch Passphrase"
                width="100%"
                height="48px"
                :on-press="onSwitch"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';

import { useNotify, useStore } from '@/utils/hooks';
import { OBJECTS_MUTATIONS } from '@/store/modules/objects';
import { AnalyticsErrorEventSource } from '@/utils/constants/analyticsEventNames';
import { MODALS } from '@/utils/constants/appStatePopUps';
import { APP_STATE_MUTATIONS } from '@/store/mutationConstants';
import { EdgeCredentials } from '@/types/accessGrants';

import VButton from '@/components/common/VButton.vue';
import VInput from '@/components/common/VInput.vue';

const props = withDefaults(defineProps<{
    onCancel?: () => void,
}>(), {
    onCancel: () => () => {},
});

const notify = useNotify();
const store = useStore();

const passphrase = ref<string>('');
const enterError = ref<string>('');

/**
 * Sets passphrase input value to local variable.
 * Resets error is present.
 * @param value
 */
function setPassphrase(value: string): void {
    if (enterError.value) {
        enterError.value = '';
    }

    passphrase.value = value;
}

/**
 * Sets new passphrase and generates new edge credentials.
 */
async function onSwitch(): Promise<void> {
    if (!passphrase.value) {
        enterError.value = 'Passphrase can\'t be empty';

        return;
    }

    try {
        store.commit(OBJECTS_MUTATIONS.SET_GATEWAY_CREDENTIALS, new EdgeCredentials());
        store.commit(OBJECTS_MUTATIONS.SET_PASSPHRASE, passphrase.value);
        store.commit(OBJECTS_MUTATIONS.SET_PROMPT_FOR_PASSPHRASE, false);

        notify.success('Passphrase was switched successfully');
        store.commit(APP_STATE_MUTATIONS.UPDATE_ACTIVE_MODAL, MODALS.manageProjectPassphrase);
    } catch (error) {
        await notify.error(error.message, AnalyticsErrorEventSource.SWITCH_PROJECT_LEVEL_PASSPHRASE_MODAL);
    }
}
</script>

<style scoped lang="scss">
.switch-step {
    display: flex;
    flex-direction: column;
    align-items: center;
    font-family: 'font_regular', sans-serif;
    max-width: 433px;

    &__title {
        font-family: 'font_bold', sans-serif;
        font-size: 32px;
        line-height: 39px;
        color: #1b2533;
        margin: 14px 0;
    }

    &__info {
        font-size: 14px;
        line-height: 19px;
        color: #354049;
        margin-bottom: 24px;
    }

    &__buttons {
        display: flex;
        align-items: center;
        justify-content: center;
        column-gap: 33px;
        margin-top: 20px;
        width: 100%;

        @media screen and (max-width: 530px) {
            column-gap: unset;
            flex-direction: column-reverse;
            row-gap: 15px;
        }
    }
}
</style>
