declare global {
  interface Window {
    Telegram?: {
      WebApp?: {
        initData: string;
        colorScheme: 'light' | 'dark';
        ready: () => void;
        expand: () => void;
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
