declare global {
  interface Window {
    /** Desktop/mobile: native app exposes this to receive method calls (e.g. web_app_expand) */
    TelegramWebviewProxy?: { postEvent: (event: string, data: string) => void };
    Telegram?: {
      WebApp?: {
        initData: string;
        /** Platform: android | ios | macos | tdesktop | weba | web */
        platform?: string;
        colorScheme: 'light' | 'dark';
        ready: () => void;
        expand: () => void;
        requestFullscreen?: () => void;
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
 * Calls the web_app_request_fullscreen method to request fullscreen for the Mini App (since v8.0).
 * Uses the official method so it works in Web (iframe), Desktop and Mobile.
 * @see https://docs.telegram-mini-apps.com/platform/methods#web-app-request-fullscreen
 */
export function requestFullscreenMiniApp(): void {
  if (typeof window === 'undefined') return;
  const proxy = window.TelegramWebviewProxy;
  if (proxy?.postEvent) {
    proxy.postEvent('web_app_request_fullscreen', '{}');
    return;
  }
  const tw = window.Telegram?.WebApp;
  if (tw?.requestFullscreen) {
    tw.requestFullscreen();
    return;
  }
  try {
    window.parent.postMessage(
      JSON.stringify({ eventType: 'web_app_request_fullscreen', eventData: {} }),
      'https://web.telegram.org'
    );
  } catch {
    // not in iframe or same-origin
  }
}

const TELEGRAM_WEB_ORIGIN = 'https://web.telegram.org';

/**
 * Sends a Telegram Mini App method. Works in Web (postMessage) and Desktop/Mobile (TelegramWebviewProxy.postEvent).
 * @see https://docs.telegram-mini-apps.com/platform/methods
 */
function postTelegramMethod(eventType: string, eventData: Record<string, unknown> = {}): void {
  if (typeof window === 'undefined') return;
  const data = JSON.stringify(eventData);
  const proxy = window.TelegramWebviewProxy;
  if (proxy?.postEvent) {
    proxy.postEvent(eventType, data);
    return;
  }
  try {
    window.parent.postMessage(
      JSON.stringify({ eventType, eventData }),
      TELEGRAM_WEB_ORIGIN
    );
  } catch {
    // not in iframe or same-origin
  }
}

/**
 * Opens a t.me link in the Telegram app via web_app_open_tg_link (Mini App will close).
 * link must be in format https://t.me/<path> (e.g. https://t.me/BotUsername/thread_id).
 * @see https://docs.telegram-mini-apps.com/platform/methods#web_app_open_tg_link
 */
export function openTelegramLink(link: string): void {
  if (typeof window === 'undefined' || !link) return;
  const prefix = 'https://t.me';
  if (!link.startsWith(prefix)) return;
  const pathFull = link.slice(prefix.length);
  if (!pathFull) return;
  postTelegramMethod('web_app_open_tg_link', { path_full: pathFull });
}

/**
 * Disables vertical swipe-to-close (since v7.7). Prevents accidental minimize when scrolling.
 * @see https://docs.telegram-mini-apps.com/platform/methods#web_app_setup_swipe_behavior
 */
export function setupSwipeBehavior(allowVerticalSwipe = false): void {
  postTelegramMethod('web_app_setup_swipe_behavior', { allow_vertical_swipe: allowVerticalSwipe });
}

/**
 * Converts a CSS color value to #RRGGBB for Telegram API.
 * getComputedStyle may return hex or rgb(r,g,b); Telegram expects #RRGGBB.
 */
function toHexColor(cssValue: string): string {
  const s = cssValue.trim();
  if (/^#[0-9A-Fa-f]{6}$/.test(s)) return s;
  const rgb = s.match(/^rgb\s*\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*\)$/);
  if (rgb) {
    const r = Number.parseInt(rgb[1], 10).toString(16).padStart(2, '0');
    const g = Number.parseInt(rgb[2], 10).toString(16).padStart(2, '0');
    const b = Number.parseInt(rgb[3], 10).toString(16).padStart(2, '0');
    return `#${r}${g}${b}`;
  }
  return s;
}

/**
 * Sets Mini App background, header, and bottom bar colors from CSS theme (--background, --muted).
 * Call after theme is applied. Ensures white theme uses #ffffff for header/bg/bottom so they match.
 * @see https://docs.telegram-mini-apps.com/platform/methods#web_app_set_background_color
 * @see https://docs.telegram-mini-apps.com/platform/methods#web_app_set_header_color
 * @see https://docs.telegram-mini-apps.com/platform/methods#web_app_set_bottom_bar_color
 */
export function setTelegramThemeColors(): void {
  if (typeof document === 'undefined') return;
  const style = getComputedStyle(document.documentElement);
  const bgRaw = style.getPropertyValue('--background').trim();
  const mutedRaw = style.getPropertyValue('--muted').trim();
  const bg = toHexColor(bgRaw);
  const bottomBar = toHexColor(mutedRaw);
  if (bg) {
    postTelegramMethod('web_app_set_background_color', { color: bg });
    postTelegramMethod('web_app_set_header_color', { color: bg });
  }
  if (bottomBar) postTelegramMethod('web_app_set_bottom_bar_color', { color: bottomBar });
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
