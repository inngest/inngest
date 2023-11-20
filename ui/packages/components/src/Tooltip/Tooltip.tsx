'use client';

import React from 'react';
import * as TooltipPrimitive from '@radix-ui/react-tooltip';

import { classNames } from '../utils/classNames';

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
    className={classNames(
      'animate-slide-down-fade shadow-floating z-50 max-w-xs rounded-md bg-white/95 px-2 py-1 text-sm text-slate-800 ring-1 ring-black/5 backdrop-blur-[3px] dark:bg-slate-400',
      className
    )}
    {...props}
  />
));

TooltipContent.displayName = TooltipPrimitive.Content.displayName;

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider };
