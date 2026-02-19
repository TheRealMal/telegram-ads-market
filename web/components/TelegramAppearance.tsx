'use client';

import { useEffect } from 'react';
import { setupSwipeBehavior, setTelegramThemeColors } from '@/lib/telegram';

/**
 * Applies Telegram Mini App appearance: disables vertical swipe and sets background/bottom bar
 * colors from the current theme (--background, --muted). Runs on mount and when theme is toggled.
 */
export function TelegramAppearance() {
  useEffect(() => {
    setupSwipeBehavior(false);
    setTelegramThemeColors();

    const onThemeChanged = () => setTelegramThemeColors();
    window.addEventListener('ads_mrkt_theme_changed', onThemeChanged);
    return () => window.removeEventListener('ads_mrkt_theme_changed', onThemeChanged);
  }, []);

  return null;
}
