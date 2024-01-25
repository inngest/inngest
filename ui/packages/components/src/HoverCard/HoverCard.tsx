import { forwardRef } from 'react';
import * as HoverCardPrimitive from '@radix-ui/react-hover-card';

import { classNames } from '../utils/classNames';

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
        className={classNames(
          className,
          'shadow-outline-primary-light rounded-md bg-white p-2 dark:bg-slate-700'
        )}
      >
        <HoverCardPrimitive.Arrow className="fill-white dark:fill-slate-700" />
        {children}
      </HoverCardPrimitive.Content>
    </HoverCardPrimitive.Portal>
  );
});
