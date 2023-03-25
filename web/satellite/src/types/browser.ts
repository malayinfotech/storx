// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

import { VueConstructor } from 'vue';

import EmailIcon from '../../static/images/objects/email.svg';

/**
 * Exposes all properties and methods present and available in the file/browser objects in Browser.
 */
export interface BrowserFile extends File {
  Key: string;
  LastModified: Date;
  Size: number;
  type: string;
}

export enum ShareOptions {
  Reddit = 'Reddit',
  Facebook = 'Facebook',
  Twitter = 'Twitter',
  HackerNews = 'Hacker News',
  LinkedIn = 'LinkedIn',
  Telegram = 'Telegram',
  WhatsApp = 'WhatsApp',
  Email = 'E-Mail',
}

export class ShareButtonConfig {
    constructor(
      public label: ShareOptions = ShareOptions.Email,
      public color: string = 'white',
      public link: string = '',
      public image: VueConstructor = EmailIcon,
    ) {}
}
