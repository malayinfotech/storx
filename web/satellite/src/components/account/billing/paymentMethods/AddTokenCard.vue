// Copyright (C) 2022 Storx Labs, Inc.
// See LICENSE for copying information.

<template>
    <div class="token">
        <div class="token__large-icon-container">
            <div class="token__large-icon">
                <StorxLarge />
            </div>
        </div>
        <div class="token__add-funds">
            <h3 class="token__add-funds__title">
                STORX Token
            </h3>
            <p class="token__add-funds__support-info">To deposit STORX token and request higher limits, please contact <a target="_blank" rel="noopener noreferrer" href="https://supportdcs.storx/hc/en-us/requests/new?ticket_form_id=360000683212">Support</a></p>
        </div>
    </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';

import StorxLarge from '@/../static/images/billing/storx-icon-large.svg';

// @vue/component
@Component({
    components: {
        StorxLarge,
    },
})
export default class AddTokenCard extends Vue {}
</script>

<style scoped lang="scss">
    .token-loader {
        width: 100% !important;
        padding: 0 !important;
        margin: 40px 0;
    }

    .token {
        border-radius: 10px;
        width: 348px;
        height: 203px;
        box-sizing: border-box;
        padding: 24px;
        box-shadow: 0 0 20px rgb(0 0 0 / 4%);
        background: #fff;
        position: relative;
        font-family: 'font_regular', sans-serif;

        &__large-icon-container {
            position: absolute;
            top: 0;
            right: 0;
            height: 120px;
            width: 120px;
            border-radius: 10px;
            overflow: hidden;
        }

        &__large-icon {
            position: absolute;
            top: -25px;
            right: -24px;
        }

        &__add-funds {
            position: relative;
            display: flex;
            flex-direction: column;
            justify-content: center;
            gap: 1rem;
            height: 100%;
            width: 100%;

            &__title {
                font-family: 'font_bold', sans-serif;
            }

            &__support-info {
                font-size: 14px;
                line-height: 20px;
                color: #000;

                a {
                    color: var(--c-blue-3);
                    text-decoration: underline !important;
                }
            }
        }
    }
</style>
