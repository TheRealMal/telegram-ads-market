'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useCallback, useEffect, useRef } from 'react';
import { Home, ListChecks, Briefcase, User } from 'lucide-react';
import { cn } from '@/lib/utils';

const navItems = [
  { id: 'market', label: 'Market', icon: Home, path: '/' },
  { id: 'listings', label: 'Listings', icon: ListChecks, path: '/my-listings' },
  { id: 'deals', label: 'Deals', icon: Briefcase, path: '/deals' },
  { id: 'profile', label: 'Profile', icon: User, path: '/profile' },
] as const;

export function BottomNav() {
  const pathname = usePathname();
  const containerRef = useRef<HTMLDivElement>(null);

  // Analytics dashboard is a direct-URL page, not in main nav
  if (pathname.startsWith('/analytics')) return null;
  const listRef = useRef<HTMLUListElement>(null);

  const isActive = useCallback(
    (item: (typeof navItems)[number]) =>
      item.path === '/'
        ? pathname === '/'
        : pathname === item.path || pathname.startsWith(item.path + '/'),
    [pathname]
  );

  const activeIndex = navItems.findIndex((item) => isActive(item));
  const safeIndex = activeIndex >= 0 ? activeIndex : 0;

  useEffect(() => {
    const list = listRef.current;
    const container = containerRef.current;
    if (!list || !container) return;

    const activeEl = list.querySelector('li.active') as HTMLElement | null;
    if (!activeEl) return;

    const listRect = list.getBoundingClientRect();
    const liRect = activeEl.getBoundingClientRect();

    container.style.setProperty('--pill-x', `${liRect.left - listRect.left}px`);
    container.style.setProperty('--pill-w', `${liRect.width}px`);
    container.style.setProperty('--pill-h', `${liRect.height}px`);
  }, [pathname, safeIndex]);

  return (
    <nav
      id="container-bottombar"
      ref={containerRef}
      className="fixed bottom-6 left-4 right-4 z-50 mx-auto flex max-w-lg items-center justify-center"
      aria-label="Main navigation"
    >
      <ul
        ref={listRef}
        className="flex h-16 w-full items-stretch rounded-full bg-white/72 dark:bg-black/48 backdrop-blur-md backdrop-saturate-150"
      >
        {navItems.map((item) => {
          const Icon = item.icon;
          const active = isActive(item);
          return (
            <li
              key={item.id}
              className={cn(
                'flex flex-1 flex-col items-center justify-center gap-0.5 transition-colors',
                active && 'active'
              )}
            >
              <Link
                href={item.path}
                className={cn(
                  'flex h-full w-full flex-col items-center justify-center gap-0.5',
                  active ? 'text-primary' : 'text-muted-foreground hover:text-foreground'
                )}
                aria-current={active ? 'page' : undefined}
              >
                <Icon size={22} strokeWidth={2} aria-hidden />
                <span className="text-[10px] font-semibold">{item.label}</span>
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
