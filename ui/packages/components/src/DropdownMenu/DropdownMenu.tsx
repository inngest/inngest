import { forwardRef } from 'react';
import * as DropdownMenuPrimitive from '@radix-ui/react-dropdown-menu';

import { cn } from '../utils/classNames';
import { menuItemClassName, menuPanelClassName, menuSeparatorClassName } from './menuStyles';

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
        className={cn(menuPanelClassName, 'z-50 min-w-40', props.className)}
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
        menuItemClassName,
        'hover:bg-canvasSubtle data-[highlighted]:bg-canvasSubtle data-[disabled]:text-disabled cursor-pointer data-[disabled]:cursor-not-allowed data-[disabled]:hover:bg-transparent',
        props.className
      )}
    >
      {children}
    </DropdownMenuPrimitive.Item>
  );
});

export const DropdownMenuSeparator = forwardRef<
  React.ElementRef<typeof DropdownMenuPrimitive.Separator>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Separator>
>((props, forwardedRef) => {
  return (
    <DropdownMenuPrimitive.Separator
      {...props}
      ref={forwardedRef}
      className={cn(menuSeparatorClassName, props.className)}
    />
  );
});
