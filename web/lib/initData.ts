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
