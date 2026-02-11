'use client';

import { useEffect } from 'react';
import { expandMiniApp } from '@/lib/telegram';

/**
 * When the mini app is opened on a phone (touch device or narrow viewport),
 * request fullscreen via the web_app_expand method.
 * On desktop we leave the app as-is.
 * @see https://docs.telegram-mini-apps.com/platform/methods#web-app-expand
 */
export function TelegramExpandOnMobile() {
  useEffect(() => {
    if (typeof window === 'undefined') return;
    const isMobile =
      /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent) ||
      'ontouchstart' in window ||
      window.innerWidth < 768;
    if (isMobile) expandMiniApp();
  }, []);
  return null;
}
