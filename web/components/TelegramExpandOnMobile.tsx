'use client';

import { useEffect } from 'react';
import { requestFullscreenMiniApp } from '@/lib/telegram';
import { isTelegramPhone } from '@/lib/initData';

/**
 * When the mini app is opened on Telegram for Android or iOS (platform from init data),
 * request fullscreen via web_app_request_fullscreen. On desktop (macos, tdesktop, weba, web) we leave as-is.
 * @see https://docs.telegram-mini-apps.com/platform/methods#web-app-request-fullscreen
 */
export function TelegramExpandOnMobile() {
  useEffect(() => {
    if (typeof window === 'undefined') return;
    if (isTelegramPhone()) requestFullscreenMiniApp();
  }, []);
  return null;
}
