// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

<template>
    <div class="add-storx-area">
        <p class="add-storx-area__support-info">To deposit STORX token and request higher limits, please contact <a target="_blank" rel="noopener noreferrer" href="https://supportdcs.storx/hc/en-us/requests/new?ticket_form_id=360000683212">Support</a></p>
    </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';

// @vue/component
@Component
export default class AddStorxForm extends Vue {}
</script>

<style scoped lang="scss">
    p {
        margin: 0;
    }

    .add-storx-area {
        margin: 20px 0;
        font-family: 'font_regular', sans-serif;
        display: flex;
        max-height: 52px;
        justify-content: space-between;
        align-items: center;

        &__selection-container {
            display: flex;
            align-items: center;

            &__label {
                margin-right: 30px;
                max-width: 215px;
            }

            &__form {
                width: 60%;
            }
        }

        &__submit-area {
            display: flex;
            align-items: center;
            min-width: 135px;
        }

        &__support-info {
            font-weight: 600;
            font-size: 14px;
            line-height: 20px;
            color: #000;

            a {
                color: var(--c-blue-3);
            }
        }
    }

    .loading-image {
        width: 18px;
        height: 18px;
        margin-right: 5px;
    }
</style>
