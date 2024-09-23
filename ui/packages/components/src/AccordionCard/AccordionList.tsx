import { forwardRef } from 'react';
import * as AccordionPrimitive from '@radix-ui/react-accordion';
import { RiArrowDownSLine } from '@remixicon/react';

export function AccordionList({
  children,
  ...props
}: React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Root>) {
  return (
    <AccordionPrimitive.Root
      className="border-muted bg-canvasBase divide-subtle divide-y overflow-hidden rounded-lg border"
      {...props}
    >
      {children}
    </AccordionPrimitive.Root>
  );
}

const AccordionItem = forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Item>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Item>
>(({ children, ...props }, forwardedRef) => {
  return (
    <AccordionPrimitive.Item
      {...props}
      ref={forwardedRef}
      className="first:rounded-t last:rounded-b"
    >
      {children}
    </AccordionPrimitive.Item>
  );
});

const AccordionTrigger = forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Trigger>
>(({ children, ...props }, forwardedRef) => {
  return (
    <AccordionPrimitive.Header className="data-[state=open]:border-muted flex items-center text-sm data-[state=open]:border-b">
      <AccordionPrimitive.Trigger
        {...props}
        ref={forwardedRef}
        className="hover:bg-canvasSubtle group w-full"
      >
        <div className="flex items-center gap-1 px-3 py-2">
          <RiArrowDownSLine className="transform-90 h-5 w-5 transition-transform duration-500 group-data-[state=open]:-rotate-180" />
          {children}
        </div>
      </AccordionPrimitive.Trigger>
    </AccordionPrimitive.Header>
  );
});

const AccordionContent = forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Content>
>(({ children, ...props }, forwardedRef) => {
  return (
    <AccordionPrimitive.Content {...props} ref={forwardedRef} className="px-4 py-3 text-sm">
      {children}
    </AccordionPrimitive.Content>
  );
});

AccordionList.Item = AccordionItem;
AccordionList.Trigger = AccordionTrigger;
AccordionList.Content = AccordionContent;
