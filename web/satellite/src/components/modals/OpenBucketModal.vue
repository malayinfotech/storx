// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

<template>
    <VModal :on-close="closeModal">
        <template #content>
            <div class="modal">
                <OpenBucketIcon />
                <h1 class="modal__title">Enter your encryption passphrase</h1>
                <p class="modal__info">
                    To open a bucket and view your encrypted files, <br>please enter your encryption passphrase.
                </p>
                <VInput
                    :class="{'orange-border': isWarningState}"
                    label="Encryption Passphrase"
                    placeholder="Enter a passphrase here"
                    :error="enterError"
                    role-description="passphrase"
                    is-password
                    :disabled="isLoading"
                    @setData="setPassphrase"
                />
                <div v-if="isWarningState" class="modal__warning">
                    <OpenWarningIcon class="modal__warning__icon" />
                    <div class="modal__warning__info">
                        <p class="modal__warning__info__title">
                            This bucket includes files that are uploaded using a different encryption passphrase from
                            the one you entered.
                        </p>
                    </div>
                </div>
                <div class="modal__buttons">
                    <VButton
                        label="Cancel"
                        height="48px"
                        :is-transparent="true"
                        :on-press="closeModal"
                        :is-disabled="isLoading"
                    />
                    <VButton
                        :label="isWarningState ? 'Continue Anyway ->' : 'Continue ->'"
                        height="48px"
                        :on-press="onContinue"
                        :is-disabled="isLoading"
                        :is-orange="isWarningState"
                    />
                </div>
            </div>
        </template>
    </VModal>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';

import { RouteConfig } from '@/router';
import { OBJECTS_ACTIONS, OBJECTS_MUTATIONS } from '@/store/modules/objects';
import { AnalyticsHttpApi } from '@/api/analytics';
import { AnalyticsErrorEventSource } from '@/utils/constants/analyticsEventNames';
import { Bucket } from '@/types/buckets';
import { MODALS } from '@/utils/constants/appStatePopUps';
import { APP_STATE_MUTATIONS } from '@/store/mutationConstants';

import VModal from '@/components/common/VModal.vue';
import VInput from '@/components/common/VInput.vue';
import VButton from '@/components/common/VButton.vue';

import OpenBucketIcon from '@/../static/images/buckets/openBucket.svg';
import OpenWarningIcon from '@/../static/images/objects/openWarning.svg';

// @vue/component
@Component({
    components: {
        VInput,
        VModal,
        VButton,
        OpenBucketIcon,
        OpenWarningIcon,
    },
})
export default class OpenBucketModal extends Vue {
    private readonly NUMBER_OF_DISPLAYED_OBJECTS = 1000;
    private readonly analytics: AnalyticsHttpApi = new AnalyticsHttpApi();

    public enterError = '';
    public passphrase = '';
    public isLoading = false;
    public isWarningState = false;

    /**
     * Sets access and navigates to object browser.
     */
    public async onContinue(): Promise<void> {
        if (this.isLoading) return;

        if (this.isWarningState) {
            this.$store.commit(OBJECTS_MUTATIONS.SET_PROMPT_FOR_PASSPHRASE, false);

            this.closeModal();
            this.analytics.pageVisit(RouteConfig.Buckets.with(RouteConfig.UploadFile).path);
            await this.$router.push(RouteConfig.Buckets.with(RouteConfig.UploadFile).path);

            return;
        }

        if (!this.passphrase) {
            this.enterError = 'Passphrase can\'t be empty';
            this.analytics.errorEventTriggered(AnalyticsErrorEventSource.OPEN_BUCKET_MODAL);

            return;
        }

        this.isLoading = true;

        try {
            this.$store.commit(OBJECTS_MUTATIONS.SET_PASSPHRASE, this.passphrase);
            await this.$store.dispatch(OBJECTS_ACTIONS.SET_S3_CLIENT);
            const count: number = await this.$store.dispatch(OBJECTS_ACTIONS.GET_OBJECTS_COUNT, this.bucketName);
            if (this.bucketObjectCount > count && this.bucketObjectCount <= this.NUMBER_OF_DISPLAYED_OBJECTS) {
                this.isWarningState = true;
                this.isLoading = false;
                return;
            }
            this.$store.commit(OBJECTS_MUTATIONS.SET_PROMPT_FOR_PASSPHRASE, false);
            this.isLoading = false;

            this.closeModal();
            this.analytics.pageVisit(RouteConfig.Buckets.with(RouteConfig.UploadFile).path);
            await this.$router.push(RouteConfig.Buckets.with(RouteConfig.UploadFile).path);
        } catch (error) {
            await this.$notify.error(error.message, AnalyticsErrorEventSource.OPEN_BUCKET_MODAL);
            this.isLoading = false;
        }
    }

    /**
     * Closes open bucket modal.
     */
    public closeModal(): void {
        if (this.isLoading) return;

        this.$store.commit(APP_STATE_MUTATIONS.UPDATE_ACTIVE_MODAL, MODALS.openBucket);
    }

    /**
     * Sets passphrase from child component.
     */
    public setPassphrase(passphrase: string): void {
        if (this.enterError) this.enterError = '';
        if (this.isWarningState) this.isWarningState = false;

        this.passphrase = passphrase;
    }

    /**
     * Returns chosen bucket name from store.
     */
    public get bucketName(): string {
        return this.$store.state.objectsModule.fileComponentBucketName;
    }

    /**
     * Returns selected bucket name object count.
     */
    private get bucketObjectCount(): number {
        const data: Bucket = this.$store.state.bucketUsageModule.page.buckets.find((bucket: Bucket) => bucket.name === this.bucketName);

        return data?.objectCount || 0;
    }
}
</script>

<style scoped lang="scss">
    .modal {
        font-family: 'font_regular', sans-serif;
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 62px 62px 54px;
        max-width: 500px;

        @media screen and (max-width: 600px) {
            padding: 62px 24px 54px;
        }

        &__title {
            font-family: 'font_bold', sans-serif;
            font-size: 26px;
            line-height: 31px;
            color: #131621;
            margin: 30px 0 15px;
        }

        &__info {
            font-size: 16px;
            line-height: 21px;
            text-align: center;
            color: #354049;
            margin-bottom: 32px;
        }

        &__warning {
            max-width: 405px;
            padding: 16px;
            display: flex;
            align-items: flex-start;
            background: #fec;
            border: 1px solid #ffd78a;
            box-shadow: 0 7px 20px rgb(0 0 0 / 15%);
            border-radius: 10px;
            margin-top: 22px;

            &__icon {
                min-width: 32px;
            }

            &__info {
                margin-left: 16px;

                &__title {
                    font-family: 'font_medium', sans-serif;
                    font-size: 14px;
                    line-height: 20px;
                    color: #000;
                    text-align: left;
                }
            }
        }

        &__buttons {
            display: flex;
            column-gap: 20px;
            margin-top: 31px;
            width: 100%;

            @media screen and (max-width: 500px) {
                flex-direction: column-reverse;
                column-gap: unset;
                row-gap: 20px;
            }
        }
    }

    .orange-border {

        :deep(input) {
            border-color: #ff8a00;
        }
    }
</style>
