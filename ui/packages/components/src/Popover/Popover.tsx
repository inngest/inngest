import { forwardRef } from 'react';
import * as PopoverPrimitive from '@radix-ui/react-popover';

export const Popover = PopoverPrimitive.Root;
export const PopoverTrigger = PopoverPrimitive.Trigger;
export const PopoverClose = PopoverPrimitive.Close;

export const PopoverContent = forwardRef<
  React.ElementRef<typeof PopoverPrimitive.Portal>,
  React.ComponentPropsWithoutRef<typeof PopoverPrimitive.Portal>
>(({ children, ...props }, forwardedRef) => (
  <PopoverPrimitive.Portal>
    <PopoverPrimitive.Content
      sideOffset={5}
      {...props}
      ref={forwardedRef}
      className="rounded bg-white drop-shadow"
    >
      {children}
    </PopoverPrimitive.Content>
  </PopoverPrimitive.Portal>
));
