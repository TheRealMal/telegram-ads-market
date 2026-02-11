import { Address } from '@ton/core';

/**
 * Converts a TON address (raw "workchain:hex" or user-friendly base64) to user-friendly format
 * suitable for display and for TON Connect sendTransaction.
 */
export function toFriendlyAddress(addr: string, bounceable: boolean = true): string {
  if (!addr || typeof addr !== 'string') return addr;
  try {
    const a = Address.parse(addr);
    return a.toString({ bounceable: bounceable, urlSafe: true });
  } catch {
    return addr;
  }
}

/**
 * Returns a display string for any TON address (friendly format, same everywhere).
 */
export function formatAddressForDisplay(addr: string): string {
  return toFriendlyAddress(addr);
}

/**
 * Truncates a TON address for compact display (e.g. "EQ...abc1").
 */
export function truncateAddressDisplay(addr: string, head = 4, tail = 4): string {
  const friendly = formatAddressForDisplay(addr);
  if (!friendly || friendly.length <= head + tail) return friendly;
  return `${friendly.slice(0, head)}...${friendly.slice(-tail)}`;
}

/**
 * Returns true if two addresses refer to the same account (handles raw vs friendly).
 */
export function addressesEqual(a: string, b: string): boolean {
  if (!a || !b) return false;
  try {
    return Address.parse(a).equals(Address.parse(b));
  } catch {
    return false;
  }
}
