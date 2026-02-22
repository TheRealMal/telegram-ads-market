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
      {/* Liquid glass top bar: Convex Squircle bottom edge (2rem), specular rim */}
      <div
        className="pointer-events-none absolute bottom-0 left-4 right-4 mx-auto h-12 max-w-lg rounded-b-[2rem] border border-t-0 border-border/50 bg-white/72 shadow-[inset_0_1px_0_0_rgba(255,255,255,0.4),0_4px_24px_-8px_rgba(0,0,0,0.08)] backdrop-blur-md backdrop-saturate-150 dark:bg-black/48 dark:shadow-[inset_0_1px_0_0_rgba(255,255,255,0.08),0_4px_24px_-8px_rgba(0,0,0,0.2)]"
        aria-hidden
      >
        <div
          className="absolute inset-0 rounded-b-[2rem] bg-gradient-to-b from-white/25 via-white/5 to-transparent dark:from-white/10 dark:via-white/[0.02] dark:to-transparent"
          aria-hidden
        />
      </div>
      <div
        className="pointer-events-none absolute bottom-0 left-0 right-0 h-10 bg-gradient-to-b from-transparent to-background"
        aria-hidden
      />
    </div>
  );
}
