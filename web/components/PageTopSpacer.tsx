'use client';

import { useEffect, useState } from 'react';

/**
 * Blank sticky top spacer for Telegram mini app (reserves space for app buttons).
 * Renders nothing when the app is already in fullscreen (isExpanded) so the empty header is hidden.
 */
export function PageTopSpacer() {
  const [hide, setHide] = useState(false);

  useEffect(() => {
    const tw = typeof window !== 'undefined' ? window.Telegram?.WebApp : undefined;
    if (!tw) return;
    const update = () => setHide(Boolean(tw?.isExpanded));
    update();
    tw?.onEvent?.('viewportChanged', update);
    const id = setInterval(update, 400);
    const t = setTimeout(() => clearInterval(id), 3000);
    return () => {
      tw?.offEvent?.('viewportChanged', update);
      clearInterval(id);
      clearTimeout(t);
    };
  }, []);

  if (hide) return null;

  return (
    <div className="sticky top-0 z-30 shrink-0 bg-background" style={{ minHeight: '5rem' }}>
      <div
        className="pointer-events-none absolute bottom-0 left-0 right-0 h-10 bg-gradient-to-b from-transparent to-background"
        aria-hidden
      />
    </div>
  );
}
