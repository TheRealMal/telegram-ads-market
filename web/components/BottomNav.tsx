'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
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
  const isActive = (item: (typeof navItems)[number]) =>
    item.path === '/'
      ? pathname === '/'
      : pathname === item.path || pathname.startsWith(item.path + '/');

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 border-t border-border bg-background">
      <div className="mx-auto max-w-lg">
        <ul className="flex items-center justify-around px-2 py-2">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = isActive(item);
            return (
              <li key={item.id} className="flex-1">
                <Link
                  href={item.path}
                  className={cn(
                    'flex w-full flex-col items-center gap-1 py-2 transition-colors',
                    active ? 'text-primary' : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  <Icon size={20} strokeWidth={2} />
                  <span className="text-xs font-medium">{item.label}</span>
                </Link>
              </li>
            );
          })}
        </ul>
      </div>
    </nav>
  );
}
