import { getColorScheme } from './telegram';

const THEME_STORAGE_KEY = 'ads_mrkt_theme';

export type Theme = 'light' | 'dark';

export function getStoredTheme(): Theme | null {
  if (typeof window === 'undefined') return null;
  const v = localStorage.getItem(THEME_STORAGE_KEY);
  if (v === 'light' || v === 'dark') return v;
  return null;
}

/** Default theme from Telegram Web App colorScheme when in Telegram, else 'light'. */
export function getDefaultTheme(): Theme {
  if (typeof window === 'undefined') return 'light';
  return getColorScheme();
}

/** Effective theme: stored preference or Web App default. */
export function getEffectiveTheme(): Theme {
  return getStoredTheme() ?? getDefaultTheme();
}

export function setStoredTheme(theme: Theme): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem(THEME_STORAGE_KEY, theme);
  document.documentElement.classList.toggle('dark', theme === 'dark');
  window.dispatchEvent(new CustomEvent('ads_mrkt_theme_changed', { detail: theme }));
}

export function getCurrentTheme(): Theme {
  if (typeof window === 'undefined') return 'light';
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light';
}

/** Apply theme on load: use stored theme if set, otherwise Web App colorScheme. */
export function applyThemeFromWebApp(): void {
  if (typeof document === 'undefined') return;
  const theme = getEffectiveTheme();
  document.documentElement.classList.toggle('dark', theme === 'dark');
}

export function toggleTheme(): Theme {
  const next: Theme = getCurrentTheme() === 'dark' ? 'light' : 'dark';
  setStoredTheme(next);
  return next;
}
