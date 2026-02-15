'use client';

/**
 * Full-viewport loading screen. Fades out smoothly when `show` becomes false.
 * Keep content in DOM but hidden (opacity-0) while loading, then show content
 * and fade out the loader so the page appears in place.
 */
export function LoadingScreen({ show = true }: { show?: boolean }) {
  return (
    <div
      className={`fixed inset-0 z-50 flex flex-col items-center justify-center bg-background transition-opacity duration-500 ease-out ${show ? 'opacity-100' : 'opacity-0 pointer-events-none'}`}
      aria-busy={show}
      aria-label="Loading"
    >
      <div className="h-10 w-10 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      <p className="mt-4 text-sm text-muted-foreground">Loading…</p>
    </div>
  );
}
