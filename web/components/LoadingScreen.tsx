'use client';

/**
 * Full-viewport loading screen. Use while initial data is loading so the user
 * sees a single loading state, then the full page appears at once (atomic).
 */
export function LoadingScreen() {
  return (
    <div
      className="fixed inset-0 z-50 flex flex-col items-center justify-center bg-background"
      aria-busy="true"
      aria-label="Loading"
    >
      <div className="h-10 w-10 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      <p className="mt-4 text-sm text-muted-foreground">Loading…</p>
    </div>
  );
}
