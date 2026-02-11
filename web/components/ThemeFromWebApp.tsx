'use client';

import { useEffect } from 'react';
import { applyThemeFromWebApp } from '@/lib/theme';

/**
 * Applies default theme from Telegram Web App colorScheme on mount.
 * If user has a stored theme we use that; otherwise we use WebApp.colorScheme.
 */
export function ThemeFromWebApp() {
  useEffect(() => {
    applyThemeFromWebApp();
  }, []);
  return null;
}
