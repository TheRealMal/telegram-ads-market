import type { ApiResponse } from '@/types';

const BASE_URL =
  typeof window !== 'undefined'
    ? (process.env.NEXT_PUBLIC_API_URL || '').replace(/\/$/, '')
    : '';
const JWT_STORAGE_KEY = 'ads_mrkt_jwt';
/** Consider token expired this many seconds before actual exp for safer refresh */
const JWT_EXPIRY_BUFFER_SEC = 60;

function getAuthToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem(JWT_STORAGE_KEY);
}

/** Decode JWT payload and check exp (seconds since epoch). Returns true if missing or expired. */
function isJwtExpired(token: string): boolean {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return true;
    const payload = parts[1];
    const decoded = JSON.parse(
      atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
    ) as { exp?: number };
    const exp = decoded.exp;
    if (typeof exp !== 'number') return false;
    return Date.now() / 1000 >= exp - JWT_EXPIRY_BUFFER_SEC;
  } catch {
    return true;
  }
}

/** Returns a valid token from storage or refreshes via auth() and stores it. */
export async function ensureValidToken(): Promise<string | null> {
  if (typeof window === 'undefined') return null;
  const token = getAuthToken();
  if (token && !isJwtExpired(token)) return token;
  const res = await auth();
  if (res.ok && res.data) {
    setAuthToken(res.data);
    return res.data;
  }
  clearAuthToken();
  return null;
}

export async function api<T>(
  path: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const url = `${BASE_URL}${path.startsWith('/') ? path : `/${path}`}`;
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };
  const token = await ensureValidToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(url, { ...options, headers });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    return {
      ok: false,
      error_code: body.error_code || 'request_failed',
      data: body.data,
    } as ApiResponse<T>;
  }
  return body as ApiResponse<T>;
}

export function setAuthToken(token: string): void {
  if (typeof window !== 'undefined') localStorage.setItem(JWT_STORAGE_KEY, token);
}

export function clearAuthToken(): void {
  if (typeof window !== 'undefined') localStorage.removeItem(JWT_STORAGE_KEY);
}

/** Telegram Mini App init data. In Telegram this comes from WebApp.initData; locally use NEXT_PUBLIC_TG_WEB_APP_DATA. */
export function getTelegramInitData(): string | null {
  if (typeof window === 'undefined') return null;
  const w = window as Window & { Telegram?: { WebApp?: { initData?: string } } };
  const fromTelegram = w.Telegram?.WebApp?.initData;
  if (fromTelegram) return fromTelegram;
  // Local dev: use env so you can auth without opening the app inside Telegram
  let fromEnv = process.env.NEXT_PUBLIC_TG_WEB_APP_DATA?.trim();
  if (!fromEnv) return null;
  // If pasted from URL fragment (#tgWebAppData=...), decode once
  try {
    if (fromEnv.includes('%')) fromEnv = decodeURIComponent(fromEnv);
  } catch {
    // keep as-is if decode fails
  }
  return fromEnv;
}

export async function auth(referrer?: number): Promise<ApiResponse<string>> {
  const initData = getTelegramInitData();
  const url = `${BASE_URL}/api/v1/market/auth`;
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(initData ? { 'X-Telegram-InitData': initData } : {}),
  };
  const res = await fetch(url, {
    method: 'POST',
    headers,
    body: JSON.stringify({ referrer: referrer ?? null }),
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    return { ok: false, error_code: body.error_code || 'auth_failed' };
  }
  return body as ApiResponse<string>;
}
