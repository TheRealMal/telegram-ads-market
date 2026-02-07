declare global {
  interface Window {
    Telegram?: {
      WebApp?: {
        initData: string;
        colorScheme: 'light' | 'dark';
        ready: () => void;
        expand: () => void;
      };
    };
  }
}

export function getColorScheme(): 'light' | 'dark' {
  if (typeof window === 'undefined') return 'light';
  return window.Telegram?.WebApp?.colorScheme || 'light';
}

export function useTelegramTheme(): void {
  if (typeof document === 'undefined') return;
  const scheme = getColorScheme();
  document.documentElement.classList.toggle('dark', scheme === 'dark');
}
