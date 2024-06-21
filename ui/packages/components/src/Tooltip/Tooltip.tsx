'use client';

import React from 'react';
import * as TooltipPrimitive from '@radix-ui/react-tooltip';

import { cn } from '../utils/classNames';

const TooltipProvider = TooltipPrimitive.Provider;

const Tooltip = TooltipPrimitive.Root;

const TooltipTrigger = TooltipPrimitive.Trigger;

const TooltipContent = React.forwardRef<
  React.ElementRef<typeof TooltipPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TooltipPrimitive.Content>
>(({ className, sideOffset = 4, ...props }, ref) => (
  <TooltipPrimitive.Content
    ref={ref}
    sideOffset={sideOffset}
    className={cn(
      'animate-slide-down-and-fade bg-canvasBase text-basis shadow-tooltip z-50 max-w-xs rounded-md px-2 py-1 text-sm shadow-md',
      className
    )}
    {...props}
  >
    {props.children}
    <TooltipPrimitive.Arrow className="fill-tooltipArrow" />
  </TooltipPrimitive.Content>
));

const TooltipArrow = TooltipPrimitive.Arrow;

TooltipContent.displayName = TooltipPrimitive.Content.displayName;

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider, TooltipArrow };
