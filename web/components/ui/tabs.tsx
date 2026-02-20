'use client';

import * as React from 'react';
import { cn } from '@/lib/utils';

interface TabsContextValue {
  value: string;
  onValueChange: (v: string) => void;
}
const TabsContext = React.createContext<TabsContextValue | null>(null);

export function Tabs({
  defaultValue,
  value: controlledValue,
  onValueChange,
  className,
  children,
  ...props
}: React.ComponentProps<'div'> & {
  defaultValue?: string;
  value?: string;
  onValueChange?: (value: string) => void;
}) {
  const [uncontrolled, setUncontrolled] = React.useState(defaultValue ?? '');
  const value = controlledValue ?? uncontrolled;
  const handleChange = React.useCallback(
    (v: string) => {
      if (controlledValue === undefined) setUncontrolled(v);
      onValueChange?.(v);
    },
    [controlledValue, onValueChange]
  );
  return (
    <TabsContext.Provider value={{ value, onValueChange: handleChange }}>
      <div className={cn('flex flex-col gap-2', className)} {...props}>
        {children}
      </div>
    </TabsContext.Provider>
  );
}

const TabsListInner = React.forwardRef<
  HTMLDivElement,
  React.ComponentProps<'div'> & { withPill?: boolean }
>(function TabsListInner({ className, children, withPill, ...props }, ref) {
  return (
    <div
      ref={ref}
      className={cn(
        'bg-muted text-muted-foreground inline-flex h-9 w-fit items-center gap-0.5 rounded-lg p-0.5',
        withPill && 'tabs-list-with-pill',
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
});

export function TabsList({
  className,
  children,
  ...props
}: React.ComponentProps<'div'> & { withPill?: boolean }) {
  return (
    <TabsListInner className={className} {...props}>
      {children}
    </TabsListInner>
  );
}

/** TabsList with sliding pill indicator; use with glass-pill styling. Must be used inside Tabs. */
export function TabsListWithPill({
  className,
  children,
  ...props
}: React.ComponentProps<'div'>) {
  const ref = React.useRef<HTMLDivElement>(null);
  const ctx = React.useContext(TabsContext);
  const value = ctx?.value;

  React.useEffect(() => {
    const el = ref.current;
    if (!el || value == null) return;
    const updatePill = () => {
      const active = el.querySelector<HTMLElement>('[data-state="active"]');
      if (!active) return;
      const listRect = el.getBoundingClientRect();
      const triggerRect = active.getBoundingClientRect();
      el.style.setProperty('--pill-x', `${triggerRect.left - listRect.left}px`);
      el.style.setProperty('--pill-w', `${triggerRect.width}px`);
      el.style.setProperty('--pill-h', `${triggerRect.height}px`);
    };
    updatePill();
    const raf = requestAnimationFrame(updatePill);
    return () => cancelAnimationFrame(raf);
  }, [value]);

  return (
    <TabsListInner ref={ref} withPill className={className} {...props}>
      {children}
    </TabsListInner>
  );
}

export function TabsTrigger({
  value,
  className,
  children,
  ...props
}: React.ComponentProps<'button'> & { value: string }) {
  const ctx = React.useContext(TabsContext);
  const isActive = ctx?.value === value;
  return (
    <button
      type="button"
      role="tab"
      aria-selected={isActive}
      data-state={isActive ? 'active' : 'inactive'}
      onClick={() => ctx?.onValueChange(value)}
      className={cn(
        'inline-flex h-8 flex-1 items-center justify-center gap-1.5 rounded-md border border-transparent px-3 text-sm font-medium whitespace-nowrap transition-colors disabled:pointer-events-none disabled:opacity-50',
        isActive
          ? 'bg-background text-foreground shadow-sm dark:border-input dark:bg-input/30'
          : 'text-foreground dark:text-muted-foreground',
        className
      )}
      {...props}
    >
      {children}
    </button>
  );
}

export function TabsContent({
  value,
  className,
  children,
  ...props
}: React.ComponentProps<'div'> & { value: string }) {
  const ctx = React.useContext(TabsContext);
  if (ctx?.value !== value) return null;
  return (
    <div className={cn('flex-1 outline-none', className)} {...props}>
      {children}
    </div>
  );
}
