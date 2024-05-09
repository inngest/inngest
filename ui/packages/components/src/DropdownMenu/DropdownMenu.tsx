'use client';

import { forwardRef } from 'react';
import * as DropdownMenuPrimitive from '@radix-ui/react-dropdown-menu';
import { RiCheckLine, RiSubtractLine } from '@remixicon/react';

import { cn } from '../utils/classNames';

export const DropdownMenu = DropdownMenuPrimitive.Root;
export const DropdownMenuTrigger = DropdownMenuPrimitive.Trigger;

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
        align="start"
        sideOffset={props.sideOffset ?? 14}
        className={cn(
          'shadow-outline-primary-light min-w-[220px] rounded-md bg-white p-2 dark:bg-slate-700',
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
        'flex select-none items-center gap-2 rounded-md p-2 text-sm text-slate-700 hover:bg-slate-100',
        props.className
      )}
    >
      {children}
    </DropdownMenuPrimitive.Item>
  );
});

export const DropdownMenuGroup = DropdownMenuPrimitive.Group;

export const DropdownMenuCheckboxItem = forwardRef<
  React.ElementRef<typeof DropdownMenuPrimitive.CheckboxItem>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.CheckboxItem>
>(({ children, ...props }, forwardedRef) => {
  return (
    <DropdownMenuPrimitive.CheckboxItem {...props} ref={forwardedRef}>
      {children}
      <DropdownMenuPrimitive.ItemIndicator>
        {props.checked === 'indeterminate' && <RiSubtractLine />}
        {props.checked === true && <RiCheckLine />}
      </DropdownMenuPrimitive.ItemIndicator>
    </DropdownMenuPrimitive.CheckboxItem>
  );
});

export const DropdownMenuRadioGroup = DropdownMenuPrimitive.RadioGroup;

export const DropdownMenuRadioItem = forwardRef<
  React.ElementRef<typeof DropdownMenuPrimitive.RadioItem>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.RadioItem>
>(({ children, ...props }, forwardedRef) => {
  return (
    <DropdownMenuPrimitive.RadioItem {...props} ref={forwardedRef}>
      {children}
      <DropdownMenuPrimitive.ItemIndicator>
        <RiCheckLine />
      </DropdownMenuPrimitive.ItemIndicator>
    </DropdownMenuPrimitive.RadioItem>
  );
});

export const DropdownMenuSeparator = DropdownMenuPrimitive.Separator;
