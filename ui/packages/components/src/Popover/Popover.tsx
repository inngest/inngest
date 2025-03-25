import { forwardRef } from 'react';
import * as PopoverPrimitive from '@radix-ui/react-popover';

import { cn } from '../utils/classNames';

export const Popover = PopoverPrimitive.Root;
export const PopoverTrigger = PopoverPrimitive.Trigger;
export const PopoverClose = PopoverPrimitive.Close;

export const PopoverContent = forwardRef<
  React.ElementRef<typeof PopoverPrimitive.Portal>,
  React.ComponentPropsWithoutRef<typeof PopoverPrimitive.Content>
>(({ children, className, ...props }, forwardedRef) => {
  const container = typeof document !== 'undefined' ? document.getElementById('modals') : undefined;
  return (
    <PopoverPrimitive.Portal container={container}>
      <PopoverPrimitive.Content
        sideOffset={5}
        ref={forwardedRef}
        className={cn(
          'bg-canvasBase border-muted shadow-primary z-[100] max-h-[var(--radix-popover-content-available-height)] overflow-y-auto overflow-x-hidden rounded border',
          className
        )}
        {...props}
      >
        {children}
      </PopoverPrimitive.Content>
    </PopoverPrimitive.Portal>
  );
});
