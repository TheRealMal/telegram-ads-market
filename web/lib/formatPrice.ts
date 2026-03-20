/** Normalized price row: ad type, duration as number string (e.g. "24"), price as number. */
export interface PriceRow {
  adType: string;
  duration: string;
  price: number;
}

/**
 * Parse listing prices from API: array of [adType, durationStr, price]
 * (e.g. [["post", "24hr", 100]]). Returns unified { adType, duration, price }[].
 */
export function parseListingPrices(prices: unknown): PriceRow[] {
  if (Array.isArray(prices)) {
    return prices
      .filter((entry): entry is [string, string, number] => Array.isArray(entry) && entry.length >= 3)
      .map(([adType, dur, p]) => ({
        adType: String(adType ?? 'post'),
        duration: String(dur ?? '').replace(/hr$/i, '').trim() || '—',
        price: Number(p),
      }))
      .filter((row) => row.duration !== '—' && !Number.isNaN(row.price));
  }
  return [];
}

/**
 * Get first price pair for deal creation: { adType, duration, price }.
 */
export function getFirstPricePair(prices: unknown): { adType: string; duration: number; price: number } | null {
  if (Array.isArray(prices) && prices.length > 0) {
    const first = prices[0];
    if (Array.isArray(first) && first.length >= 3) {
      const adType = String(first[0] ?? 'post');
      const durStr = String(first[1] ?? '24hr');
      return {
        adType,
        duration: parseInt(durStr.replace(/\D/g, ''), 10) || 24,
        price: Number(first[2]),
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
 * Human-readable label for an ad type string.
 */
export function formatAdTypeLabel(adType: string): string {
  switch (adType) {
    case 'instant_post': return 'Instant';
    case 'story': return 'Story';
    default: return 'Post';
  }
}

/**
 * Single price entry: "24 hours - 100 TON".
 */
export function formatPriceEntry(durationStr: string, price: number): string {
  const dur = formatPriceKey(durationStr);
  return `${dur} - ${formatPriceValue(price)}`;
}
