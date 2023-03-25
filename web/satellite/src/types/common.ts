// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

export enum SortDirection {
    ASCENDING = 1,
    DESCENDING,
}

export enum OnboardingOS {
    WINDOWS = 'windows',
    MAC = 'macos',
    LINUX = 'linux',
}

export class PartneredSatellite {
    constructor(
        public name: string = '',
        public address: string = '',
    ) {}
}

export class PricingPlanInfo {
    constructor(
        public type: PricingPlanType = PricingPlanType.FREE,
        // Info for the pricing plan container
        public title: string = '',
        public containerSubtitle: string = '',
        public containerDescription: string = '',
        public containerFooterHTML: string | null = null,
        public activationButtonText: string | null = null,
        // Info for the pricing plan modal (pre-activation)
        public activationSubtitle: string | null = null,
        public activationDescriptionHTML: string = '',
        public activationPriceHTML: string | null = null,
        // Info for the pricing plan modal (post-activation)
        public successSubtitle: string = '',
    ) {}
}

export enum PricingPlanType {
    FREE = 'free',
    PARTNER = 'partner',
    PRO = 'pro',
}

// TODO: fully implement these types and their methods according to their Go counterparts
export type UUID = string
export type MemorySize = string
export type Time = string
