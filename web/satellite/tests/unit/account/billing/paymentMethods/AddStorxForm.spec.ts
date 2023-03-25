// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

import { mount } from '@vue/test-utils';

import AddStorxForm from '@/components/account/billing/paymentMethods/AddStorxForm.vue';

describe('AddStorxForm', () => {
    it('renders correctly', () => {
        const wrapper = mount<AddStorxForm>(AddStorxForm);

        expect(wrapper).toMatchSnapshot();
    });
});
