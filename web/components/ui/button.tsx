import * as React from 'react';
import { cn } from '@/lib/utils';

type ButtonVariant = 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link';
type ButtonSize = 'default' | 'sm' | 'lg' | 'icon';

const variantClasses: Record<ButtonVariant, string> = {
  default: 'bg-primary text-primary-foreground shadow-sm hover:bg-primary/90',
  destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90',
  outline:
    'border bg-background shadow-sm hover:bg-accent hover:text-accent-foreground dark:bg-input/30 dark:border-input dark:hover:bg-input/50',
  secondary: 'bg-secondary text-secondary-foreground shadow-sm hover:bg-secondary/80',
  ghost: 'hover:bg-accent hover:text-accent-foreground',
  link: 'text-primary underline-offset-4 hover:underline',
};

const sizeClasses: Record<ButtonSize, string> = {
  default: 'h-9 px-4 py-2',
  sm: 'h-8 rounded-md px-3',
  lg: 'h-10 rounded-md px-6',
  icon: 'size-9',
};

export interface ButtonProps extends React.ComponentProps<'button'> {
  variant?: ButtonVariant;
  size?: ButtonSize;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = 'default', size = 'default', ...props }, ref) => {
    return (
      <button
        ref={ref}
        className={cn(
          'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors disabled:pointer-events-none disabled:opacity-50 outline-none focus-visible:ring-2 focus-visible:ring-ring',
          variantClasses[variant],
          sizeClasses[size],
          className
        )}
        {...props}
      />
    );
  }
);
Button.displayName = 'Button';

export { Button };
