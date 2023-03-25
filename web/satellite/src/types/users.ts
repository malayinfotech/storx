// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

/**
 * Exposes all user-related functionality.
 */
export interface UsersApi {
    /**
     * Updates users full name and short name.
     *
     * @param user - contains information that should be updated
     * @throws Error
     */
    update(user: UpdatedUser): Promise<void>;
    /**
     * Fetch user.
     *
     * @returns User
     * @throws Error
     */
    get(): Promise<User>;
    /**
     * Fetches user frozen status.
     *
     * @returns boolean
     * @throws Error
     */
    getFrozenStatus(): Promise<boolean>;

    /**
     * Enable user's MFA.
     *
     * @throws Error
     */
    enableUserMFA(passcode: string): Promise<void>;
    /**
     * Disable user's MFA.
     *
     * @throws Error
     */
    disableUserMFA(passcode: string, recoveryCode: string): Promise<void>;
    /**
     * Generate user's MFA secret.
     *
     * @throws Error
     */
    generateUserMFASecret(): Promise<string>;
    /**
     * Generate user's MFA recovery codes.
     *
     * @throws Error
     */
    generateUserMFARecoveryCodes(): Promise<string[]>;
}

/**
 * User class holds info for User entity.
 */
export class User {
    public constructor(
        public id: string = '',
        public fullName: string = '',
        public shortName: string = '',
        public email: string = '',
        public partner: string = '',
        public password: string = '',
        public projectLimit: number = 0,
        public paidTier: boolean = false,
        public isMFAEnabled: boolean = false,
        public isProfessional: boolean = false,
        public position: string = '',
        public companyName: string = '',
        public employeeCount: string = '',
        public haveSalesContact: boolean = false,
        public mfaRecoveryCodeCount: number = 0,
        public _createdAt: string | null = null,
        public signupPromoCode: string = '',
        public isFrozen: boolean = false,
    ) {}

    public get createdAt(): Date | null {
        if (!this._createdAt) {
            return null;
        }
        const date = new Date(this._createdAt);
        if (date.toString().includes('Invalid')) {
            return null;
        }
        return date;
    }

    public getFullName(): string {
        return !this.shortName ? this.fullName : this.shortName;
    }
}

/**
 * User class holds info for updating User.
 */
export class UpdatedUser {
    public constructor(
        public fullName: string = '',
        public shortName: string = '',
    ) {}

    public setFullName(value: string): void {
        this.fullName = value.trim();
    }

    public setShortName(value: string): void {
        this.shortName = value.trim();
    }

    public isValid(): boolean {
        return !!this.fullName;
    }
}

/**
 * DisableMFARequest represents a request to disable multi-factor authentication.
 */
export class DisableMFARequest {
    public constructor(
        public passcode: string = '',
        public recoveryCode: string = '',
    ) {}
}

/**
 * TokenInfo represents an authentication token response.
 */
export class TokenInfo {
    public constructor(
        public token: string,
        public expiresAt: Date,
    ) {}
}
