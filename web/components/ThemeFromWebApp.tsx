'use client';

import { useEffect } from 'react';
import { applyThemeFromWebApp } from '@/lib/theme';
import { setTelegramThemeColors } from '@/lib/telegram';

/**
 * Applies default theme from Telegram Web App colorScheme on mount.
 * If user has a stored theme we use that; otherwise we use WebApp.colorScheme.
 * Then syncs Telegram header/background/bottom bar colors (e.g. white for light theme).
 */
export function ThemeFromWebApp() {
  useEffect(() => {
    applyThemeFromWebApp();
    setTelegramThemeColors();
  }, []);
  return null;
}
