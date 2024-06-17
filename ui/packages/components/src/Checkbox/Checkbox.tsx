'use client';

import * as CheckboxPrimitive from '@radix-ui/react-checkbox';
import { RiCheckLine } from '@remixicon/react';

import { cn } from '../utils/classNames';

export function Checkbox({
  className,
  ...props
}: React.ComponentProps<typeof CheckboxPrimitive.Root>) {
  return (
    <CheckboxPrimitive.Root
      className={cn(
        'data-[state=checked]:border-primary-moderate data-[state=checked]:bg-primary-moderate hover:border-contrast border-muted bg-canvasSubtle disabled:border-muted disabled:bg-subtle flex h-5 w-5 items-center justify-center rounded border outline-none transition-all',
        className
      )}
      {...props}
    >
      <CheckboxPrimitive.Indicator>
        <RiCheckLine className="text-alwaysWhite h-4 w-4" />
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  );
}

// Wrapper when using checkbox with label
function CheckboxWrapper({ children }: React.PropsWithChildren) {
  return <div className="items-top flex gap-2">{children}</div>;
}

function Label({
  children,
  ...props
}: React.PropsWithChildren<React.HTMLAttributes<HTMLLabelElement>>) {
  return (
    <label className="text-basis text-sm" {...props}>
      {children}
    </label>
  );
}

function Description({ children }: React.PropsWithChildren) {
  return <span className="text-subtle block pt-0.5">{children}</span>;
}

Checkbox.Wrapper = CheckboxWrapper;
Checkbox.Label = Label;
Checkbox.Description = Description;
