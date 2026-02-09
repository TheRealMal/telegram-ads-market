'use client';

/**
 * Blank sticky top spacer for Telegram mini app (reserves space for app buttons).
 * No header text or borders. White bottom shadow for smooth transition as list content scrolls underneath.
 */
export function PageTopSpacer() {
  return (
    <div className="sticky top-0 z-30 shrink-0 bg-background" style={{ minHeight: '5rem' }}>
      {/* Gradient fade so list items transition smoothly when scrolling underneath */}
      <div
        className="pointer-events-none absolute bottom-0 left-0 right-0 h-10 bg-gradient-to-b from-transparent to-background"
        aria-hidden
      />
    </div>
  );
}
