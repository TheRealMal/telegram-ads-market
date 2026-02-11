import { getTelegramInitData } from './api';

/** User object from Telegram Web App init data (user= param, JSON). */
export interface TelegramWebAppUser {
  id: number;
  first_name: string;
  last_name?: string;
  username?: string;
  language_code?: string;
  is_premium?: boolean;
  allows_write_to_pm?: boolean;
  photo_url?: string;
}

/** Parse init data string (key=value&...) and return the user object if present. */
export function parseInitDataUser(initData: string | null): TelegramWebAppUser | null {
  if (!initData || !initData.trim()) return null;
  try {
    const params = new URLSearchParams(initData);
    const userStr = params.get('user');
    if (!userStr) return null;
    const user = JSON.parse(userStr) as TelegramWebAppUser;
    return user && typeof user.id === 'number' ? user : null;
  } catch {
    return null;
  }
}

/** Get current user from Telegram init data (or env fallback). */
export function getTelegramUser(): TelegramWebAppUser | null {
  const data = getTelegramInitData();
  return parseInitDataUser(data);
}

/**
 * Telegram platform from WebApp: android | ios | macos | tdesktop | weba | web.
 * Use to detect phones (android, ios) vs desktop (macos, tdesktop, weba, web).
 */
export function getTelegramPlatform(): string | null {
  if (typeof window === 'undefined') return null;
  const platform = (window as Window & { Telegram?: { WebApp?: { platform?: string } } }).Telegram?.WebApp?.platform;
  return platform && typeof platform === 'string' ? platform : null;
}

/** True when running inside Telegram on a phone (Android or iOS). */
export function isTelegramPhone(): boolean {
  const p = getTelegramPlatform();
  return p === 'android' || p === 'ios';
}
