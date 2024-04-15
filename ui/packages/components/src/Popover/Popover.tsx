import { forwardRef } from 'react';
import * as PopoverPrimitive from '@radix-ui/react-popover';

export const Popover = PopoverPrimitive.Root;
export const PopoverTrigger = PopoverPrimitive.Trigger;
export const PopoverClose = PopoverPrimitive.Close;

export const PopoverContent = forwardRef<
  React.ElementRef<typeof PopoverPrimitive.Portal>,
  React.ComponentPropsWithoutRef<typeof PopoverPrimitive.Portal>
>(({ children, ...props }, forwardedRef) => {
  const container = document.getElementById('modals');
  return (
    <PopoverPrimitive.Portal container={container}>
      <PopoverPrimitive.Content
        sideOffset={5}
        {...props}
        ref={forwardedRef}
        className="rounded bg-white drop-shadow"
      >
        {children}
      </PopoverPrimitive.Content>
    </PopoverPrimitive.Portal>
  );
});
