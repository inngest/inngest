import { forwardRef } from 'react';
import * as ToggleGroupPrimitive from '@radix-ui/react-toggle-group';

import { cn } from '../utils/classNames';

export default function ToggleGroup({
  children,
  className,
  size = 'medium',
  ...props
}: React.ComponentPropsWithoutRef<typeof ToggleGroupPrimitive.Root> & {
  size?: 'small' | 'medium';
}) {
  return (
    <ToggleGroupPrimitive.Root
      className={cn(
        'bg-canvasBase border-subtle divide-muted box-border inline-flex divide-x overflow-hidden rounded-md border',
        size === 'small' ? 'h-8' : 'h-10',
        className
      )}
      {...props}
    >
      {children}
    </ToggleGroupPrimitive.Root>
  );
}

const ToggleGroupItem = forwardRef<
  React.ElementRef<typeof ToggleGroupPrimitive.Item>,
  React.ComponentPropsWithoutRef<typeof ToggleGroupPrimitive.Item>
>(({ className, ...props }, ref) => (
  <ToggleGroupPrimitive.Item
    ref={ref}
    className={cn(
      'bg-canvasBase text-muted hover:bg-primary-intense hover:text-alwaysWhite data-[state=on]:bg-primary-subtle data-[state=on]:text-alwaysWhite flex items-center justify-center px-2 text-sm first:rounded-l last:rounded-r focus:z-10 focus:outline-none',
      className
    )}
    {...props}
  >
    {props.children}
  </ToggleGroupPrimitive.Item>
));

ToggleGroup.Item = ToggleGroupItem;
