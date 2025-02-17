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

export function LabeledCheckbox({
  className,
  id,
  label,
  description,
  ...props
}: React.ComponentProps<typeof CheckboxPrimitive.Root> & {
  label: React.ReactNode;
  description?: React.ReactNode;
}) {
  return (
    <div className="items-top flex gap-2">
      <CheckboxPrimitive.Root
        className={cn(
          'data-[state=checked]:border-primary-moderate data-[state=checked]:bg-primary-moderate hover:border-contrast border-muted bg-canvasSubtle disabled:border-muted disabled:bg-subtle flex h-5 w-5 items-center justify-center rounded border outline-none transition-all',
          className
        )}
        id={id}
        {...props}
      >
        <CheckboxPrimitive.Indicator>
          <RiCheckLine className="text-alwaysWhite h-4 w-4" />
        </CheckboxPrimitive.Indicator>
      </CheckboxPrimitive.Root>
      <label className="text-basis text-sm" htmlFor={id}>
        {label}
        {description && <span className="text-muted block pt-0.5">{description}</span>}
      </label>
    </div>
  );
}
