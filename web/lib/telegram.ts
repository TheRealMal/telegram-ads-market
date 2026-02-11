declare global {
  interface Window {
    /** Desktop/mobile: native app exposes this to receive method calls (e.g. web_app_expand) */
    TelegramWebviewProxy?: { postEvent: (event: string, data: string) => void };
    Telegram?: {
      WebApp?: {
        initData: string;
        colorScheme: 'light' | 'dark';
        ready: () => void;
        expand: () => void;
        isExpanded?: boolean;
        viewportHeight?: number;
        viewportStableHeight?: number;
        onEvent?: (eventType: string, callback: () => void) => void;
        offEvent?: (eventType: string, callback: () => void) => void;
        BackButton?: {
          show: () => void;
          hide: () => void;
          onClick: (callback: () => void) => void;
          offClick?: (callback: () => void) => void;
        };
      };
    };
  }
}

/**
 * Calls the web_app_expand method to expand the Mini App to fullscreen.
 * Uses the official method name so it works in Web (iframe), Desktop and Mobile.
 * @see https://docs.telegram-mini-apps.com/platform/methods#web-app-expand
 */
export function expandMiniApp(): void {
  if (typeof window === 'undefined') return;
  const proxy = window.TelegramWebviewProxy;
  if (proxy?.postEvent) {
    proxy.postEvent('web_app_expand', '{}');
    return;
  }
  const tw = window.Telegram?.WebApp;
  if (tw?.expand) {
    tw.expand();
    return;
  }
  try {
    window.parent.postMessage(
      JSON.stringify({ eventType: 'web_app_expand', eventData: {} }),
      'https://web.telegram.org'
    );
  } catch {
    // not in iframe or same-origin
  }
}

export function getColorScheme(): 'light' | 'dark' {
  if (typeof window === 'undefined') return 'light';
  return window.Telegram?.WebApp?.colorScheme || 'light';
}

export function useTelegramTheme(): void {
  if (typeof document === 'undefined') return;
  const scheme = getColorScheme();
  document.documentElement.classList.toggle('dark', scheme === 'dark');
}

import { useEffect, useRef } from 'react';

/**
 * Shows Telegram's native Back Button while the component is mounted.
 * On back_button_pressed the onBack callback is called (e.g. router.back()).
 * Hides the button and removes the listener on unmount.
 * @see https://docs.telegram-mini-apps.com/platform/methods#web_app_setup_back_button
 */
export function useTelegramBackButton(onBack: () => void): void {
  const onBackRef = useRef(onBack);
  onBackRef.current = onBack;

  useEffect(() => {
    if (typeof window === 'undefined') return;
    const back = window.Telegram?.WebApp?.BackButton;
    if (!back) return;
    back.show();
    const handler = () => onBackRef.current();
    back.onClick(handler);
    return () => {
      back.offClick?.(handler);
      back.hide();
    };
  }, []);
}
