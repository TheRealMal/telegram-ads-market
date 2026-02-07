import * as React from 'react';
import { cn } from '@/lib/utils';

type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

const variantClasses: Record<BadgeVariant, string> = {
  default: 'border-transparent bg-primary text-primary-foreground',
  secondary: 'border-transparent bg-secondary text-secondary-foreground',
  destructive: 'border-transparent bg-destructive text-destructive-foreground',
  outline: 'text-foreground border-border',
};

export interface BadgeProps extends React.ComponentProps<'span'> {
  variant?: BadgeVariant;
}

function Badge({ className, variant = 'default', ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center justify-center rounded-md border px-2 py-0.5 text-xs font-medium transition-colors',
        variantClasses[variant],
        className
      )}
      {...props}
    />
  );
}

export { Badge };
