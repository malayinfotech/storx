// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

import { Notificator } from '@/utils/plugins/notificator';

declare module 'vue/types/vue' {
    interface Vue {
        $notify: Notificator;
    }
}
