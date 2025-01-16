'use client';

import { forwardRef } from 'react';
import * as DropdownMenuPrimitive from '@radix-ui/react-dropdown-menu';

import { cn } from '../utils/classNames';

export const DropdownMenu = DropdownMenuPrimitive.Root;

export const DropdownMenuTrigger = forwardRef<
  React.ElementRef<typeof DropdownMenuPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Trigger>
>(({ children, ...props }, forwardedRef) => {
  return (
    <DropdownMenuPrimitive.Trigger
      {...props}
      ref={forwardedRef}
      className={cn('data-[state=open]:border-primary-intense', props.className)}
    >
      {children}
    </DropdownMenuPrimitive.Trigger>
  );
});

export const DropdownMenuContent = forwardRef<
  React.ElementRef<typeof DropdownMenuPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Content>
>(({ children, ...props }, forwardedRef) => {
  return (
    <DropdownMenuPrimitive.Portal>
      <DropdownMenuPrimitive.Content
        {...props}
        ref={forwardedRef}
        onCloseAutoFocus={(event) => event.preventDefault()}
        align={props.align ?? 'start'}
        collisionPadding={8}
        sideOffset={props.sideOffset ?? 8}
        className={cn(
          'shadow-primary bg-canvasBase border-muted z-50 min-w-40 rounded-md border p-0.5 [&>*:not(:last-child)]:mb-0.5',
          props.className
        )}
      >
        {children}
      </DropdownMenuPrimitive.Content>
    </DropdownMenuPrimitive.Portal>
  );
});

export const DropdownMenuLabel = DropdownMenuPrimitive.Label;

export const DropdownMenuItem = forwardRef<
  React.ElementRef<typeof DropdownMenuPrimitive.Item>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Item>
>(({ children, ...props }, forwardedRef) => {
  return (
    <DropdownMenuPrimitive.Item
      {...props}
      ref={forwardedRef}
      className={cn(
        'text-muted hover:bg-canvasSubtle flex cursor-pointer select-none items-center gap-2 rounded-md p-2 text-[0.8125rem]',
        props.className
      )}
    >
      {children}
    </DropdownMenuPrimitive.Item>
  );
});
