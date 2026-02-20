/** Normalized price row: duration as number string (e.g. "24"), price as number. */
export interface PriceRow {
  duration: string;
  price: number;
}

/**
 * Parse listing prices from API: array of [durationStr, price] (e.g. [["24hr", 100]])
 * or object like { "24hr": 100 }. Returns unified { duration, price }[].
 */
export function parseListingPrices(prices: unknown): PriceRow[] {
  if (Array.isArray(prices)) {
    return prices
      .filter((entry): entry is [string, number] => Array.isArray(entry) && entry.length >= 2)
      .map(([dur, p]) => ({
        duration: String(dur ?? '').replace(/hr$/i, '').trim() || '—',
        price: Number(p),
      }))
      .filter((row) => row.duration !== '—' && !Number.isNaN(row.price));
  }
  if (prices && typeof prices === 'object' && !Array.isArray(prices)) {
    return Object.entries(prices).map(([k, v]) => ({
      duration: String(k).replace(/hr$/i, '').trim() || '—',
      price: Number(v),
    })).filter((row) => row.duration !== '—' && !Number.isNaN(row.price));
  }
  return [];
}

/**
 * Get first price pair for deal creation: { type, duration, price }.
 */
export function getFirstPricePair(prices: unknown): { type: string; duration: number; price: number } | null {
  if (Array.isArray(prices) && prices.length > 0) {
    const first = prices[0];
    if (Array.isArray(first) && first.length >= 2) {
      const type = String(first[0] ?? '24hr');
      return {
        type,
        duration: parseInt(type.replace(/\D/g, ''), 10) || 24,
        price: Number(first[1]),
      };
    }
  }
  if (prices && typeof prices === 'object' && !Array.isArray(prices)) {
    const ent = Object.entries(prices)[0];
    if (ent) {
      const type = String(ent[0] ?? '24hr');
      return {
        type,
        duration: parseInt(type.replace(/\D/g, ''), 10) || 24,
        price: Number(ent[1]),
      };
    }
  }
  return null;
}

/**
 * Format price key (e.g. "24hr", "48hr") as "24 hours" or "1 hour".
 */
export function formatPriceKey(key: string): string {
  const trimmed = (key || '').trim().replace(/hr$/i, '');
  const num = parseInt(trimmed.replace(/\D/g, ''), 10);
  if (Number.isNaN(num)) return trimmed || '—';
  return num === 1 ? '1 hour' : `${num} hours`;
}

/**
 * Format price value as "X TON" or "X.Y TON" when fractional.
 */
export function formatPriceValue(value: number): string {
  if (value == null || Number.isNaN(Number(value))) return '—';
  return `${Number(value)} TON`;
}

/**
 * Single price entry: "24 hours - 100 TON".
 */
export function formatPriceEntry(durationStr: string, price: number): string {
  const dur = formatPriceKey(durationStr);
  return `${dur} - ${formatPriceValue(price)}`;
}
