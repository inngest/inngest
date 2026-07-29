import { forwardRef, type HTMLAttributes } from 'react';
import * as SwitchPrimitive from '@radix-ui/react-switch';

import { cn } from '../utils/classNames';

export type SwitchSize = 'base' | 'sm';

// Track/thumb dimensions are coupled: the checked translate must equal
// (track width - thumb width - start offset) or the thumb over/undershoots.
const switchSizes: Record<SwitchSize, { root: string; thumb: string }> = {
  base: {
    root: 'h-6 w-[42px]',
    thumb: 'h-5 w-5 translate-x-0.5 data-[state=checked]:translate-x-[19px]',
  },
  sm: {
    root: 'h-4 w-7',
    thumb: 'h-3 w-3 translate-x-0.5 data-[state=checked]:translate-x-[14px]',
  },
};

export const Switch = forwardRef<
  React.ElementRef<typeof SwitchPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SwitchPrimitive.Root> & {
    // Tints the checked-state background. Falls back to the default green when
    // omitted. Threaded through a CSS var so it overrides the base class
    // reliably (a second bg-* class in the string would race on source order).
    checkedColor?: string;
    size?: SwitchSize;
  }
>(({ className, checkedColor, size = 'base', style, ...props }, forwardedRef) => {
  const sizing = switchSizes[size];
  return (
    <SwitchPrimitive.Root
      {...props}
      ref={forwardedRef}
      style={
        checkedColor
          ? ({ ...style, '--switch-checked-bg': checkedColor } as React.CSSProperties)
          : style
      }
      className={cn(
        'bg-surfaceMuted relative cursor-default rounded-full outline-none',
        sizing.root,
        checkedColor
          ? 'data-[state=checked]:bg-[var(--switch-checked-bg)]'
          : 'data-[state=checked]:bg-primary-moderate',
        className
      )}
    >
      <SwitchPrimitive.Thumb
        className={cn(
          'bg-alwaysWhite block rounded-full transition-transform duration-100 will-change-transform',
          sizing.thumb
        )}
      />
    </SwitchPrimitive.Root>
  );
});

export const SwitchWrapper = ({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) => <div className={cn('flex items-center gap-2', className)}>{children}</div>;

interface SwitchLabelProps extends HTMLAttributes<HTMLLabelElement> {
  htmlFor: string;
}

export const SwitchLabel = forwardRef<HTMLLabelElement, SwitchLabelProps>(
  ({ htmlFor, children, className }, ref) => {
    return (
      <label
        ref={ref}
        htmlFor={htmlFor}
        className={cn('text-basis cursor-default font-medium', className)}
      >
        {children}
      </label>
    );
  }
);
