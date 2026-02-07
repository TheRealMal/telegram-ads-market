import type { ApiResponse } from '@/types';

const BASE_URL =
  typeof window !== 'undefined'
    ? (process.env.NEXT_PUBLIC_API_URL || '').replace(/\/$/, '')
    : '';

function getAuthToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('ads_mrkt_jwt');
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
  const token = getAuthToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(url, { ...options, headers });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    return {
      ok: false,
      error_code: body.error_code || 'request_failed',
    };
  }
  return body as ApiResponse<T>;
}

export function setAuthToken(token: string): void {
  if (typeof window !== 'undefined') localStorage.setItem('ads_mrkt_jwt', token);
}

export function clearAuthToken(): void {
  if (typeof window !== 'undefined') localStorage.removeItem('ads_mrkt_jwt');
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
