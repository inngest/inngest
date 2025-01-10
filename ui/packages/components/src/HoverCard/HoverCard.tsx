import { forwardRef } from 'react';
import * as HoverCardPrimitive from '@radix-ui/react-hover-card';

import { cn } from '../utils/classNames';

export const HoverCardRoot = HoverCardPrimitive.Root;
export const HoverCardTrigger = HoverCardPrimitive.Trigger;

export const HoverCardContent = forwardRef<
  React.ElementRef<typeof HoverCardPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof HoverCardPrimitive.Content>
>(({ children, className, ...props }, forwardedRef) => {
  return (
    <HoverCardPrimitive.Portal>
      <HoverCardPrimitive.Content
        {...props}
        ref={forwardedRef}
        align="start"
        sideOffset={5}
        className={cn(className, 'shadow-primary bg-canvasBase rounded-md p-2')}
      >
        <HoverCardPrimitive.Arrow className="tooltipArrow" />
        {children}
      </HoverCardPrimitive.Content>
    </HoverCardPrimitive.Portal>
  );
});
