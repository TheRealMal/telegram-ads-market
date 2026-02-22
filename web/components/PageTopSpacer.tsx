'use client';

import { useEffect, useState } from 'react';
import { isTelegramPhone } from '@/lib/initData';

/**
 * Blank sticky top spacer for Telegram mini app. Shown only on Telegram for Android/iOS
 * (same condition as fullscreen request), so we don't add an empty header on desktop/web.
 */
export function PageTopSpacer() {
  const [show, setShow] = useState(false);

  useEffect(() => {
    setShow(isTelegramPhone());
  }, []);

  if (!show) return null;

  return (
    <div className="sticky top-0 z-30 shrink-0 bg-background" style={{ minHeight: '5.25rem' }}>
      <div
        className="pointer-events-none absolute bottom-0 left-0 right-0 h-10 bg-gradient-to-b from-transparent to-background"
        aria-hidden
      />
    </div>
  );
}
