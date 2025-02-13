import { forwardRef } from 'react';
import * as AccordionPrimitive from '@radix-ui/react-accordion';
import { RiArrowDownSLine } from '@remixicon/react';

import { cn } from '../utils/classNames';

export function AccordionList({
  children,
  className,
  ...props
}: React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Root>) {
  return (
    <AccordionPrimitive.Root
      className={cn(
        'border-subtle bg-canvasBase divide-subtle divide-y overflow-hidden rounded-md border',
        className
      )}
      {...props}
    >
      {children}
    </AccordionPrimitive.Root>
  );
}

const AccordionItem = forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Item>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Item>
>(({ children, className, ...props }, forwardedRef) => {
  return (
    <AccordionPrimitive.Item
      {...props}
      ref={forwardedRef}
      className={cn('first:rounded-t last:rounded-b', className)}
    >
      {children}
    </AccordionPrimitive.Item>
  );
});

const AccordionTrigger = forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Trigger>
>(({ children, className, ...props }, forwardedRef) => {
  return (
    <AccordionPrimitive.Header
      className={cn(
        'data-[state=open]:border-subtle flex items-center text-sm data-[state=open]:border-b',
        className
      )}
    >
      <AccordionPrimitive.Trigger
        {...props}
        ref={forwardedRef}
        className="hover:bg-canvasSubtle group w-full"
      >
        <div className="flex items-center gap-1 px-3 py-4">
          <RiArrowDownSLine className="transform-90 duration-50 h-5 w-5 transition-transform group-data-[state=open]:-rotate-180" />
          {children}
        </div>
      </AccordionPrimitive.Trigger>
    </AccordionPrimitive.Header>
  );
});

const AccordionContent = forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Content>
>(({ children, className, ...props }, forwardedRef) => {
  return (
    <AccordionPrimitive.Content
      {...props}
      ref={forwardedRef}
      className={cn('px-4 py-3 text-sm', className)}
    >
      {children}
    </AccordionPrimitive.Content>
  );
});

AccordionList.Item = AccordionItem;
AccordionList.Trigger = AccordionTrigger;
AccordionList.Content = AccordionContent;

export * as AccordionPrimitive from '@radix-ui/react-accordion';
