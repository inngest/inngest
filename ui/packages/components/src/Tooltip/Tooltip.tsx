'use client';

import React from 'react';
import * as TooltipPrimitive from '@radix-ui/react-tooltip';

import { cn } from '../utils/classNames';

const TooltipProvider = TooltipPrimitive.Provider;

const Tooltip = TooltipPrimitive.Root;

const TooltipTrigger = TooltipPrimitive.Trigger;

interface TooltipContentProps
  extends React.ComponentPropsWithoutRef<typeof TooltipPrimitive.Content> {
  hasArrow?: boolean;
}

const TooltipContent = React.forwardRef<
  React.ElementRef<typeof TooltipPrimitive.Content>,
  TooltipContentProps
>(({ className, sideOffset = 4, hasArrow = true, ...props }, ref) => (
  <TooltipPrimitive.Portal>
    <TooltipPrimitive.Content
      {...props}
      ref={ref}
      sideOffset={sideOffset}
      className={cn(
        'animate-slide-down-and-fade bg-contrast text-onContrast z-[52] max-w-xs rounded-md px-2 py-1 text-sm',
        className
      )}
    >
      {props.children}
      {hasArrow && <TooltipPrimitive.Arrow className="fill-contrast" />}
    </TooltipPrimitive.Content>
  </TooltipPrimitive.Portal>
));

const TooltipArrow = TooltipPrimitive.Arrow;

TooltipContent.displayName = TooltipPrimitive.Content.displayName;

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider, TooltipArrow };
