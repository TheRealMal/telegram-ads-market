'use client';

import { useEffect } from 'react';

/**
 * When the mini app is opened on a phone (touch device or narrow viewport), request fullscreen via Telegram.WebApp.expand().
 * On desktop we leave the app as-is.
 */
export function TelegramExpandOnMobile() {
  useEffect(() => {
    if (typeof window === 'undefined') return;
    const tw = window.Telegram?.WebApp;
    if (!tw?.expand) return;
    const isMobile =
      /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent) ||
      'ontouchstart' in window ||
      window.innerWidth < 768;
    if (isMobile) tw.expand();
  }, []);
  return null;
}
