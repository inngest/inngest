import { forwardRef } from 'react';
import { cn } from '@inngest/components/utils/classNames';
import * as TabsPrimitive from '@radix-ui/react-tabs';

export default function TabCards({
  children,
  ...props
}: React.ComponentPropsWithoutRef<typeof TabsPrimitive.Root>) {
  return <TabsPrimitive.Root {...props}>{children}</TabsPrimitive.Root>;
}

const TabButton = forwardRef<
  React.ElementRef<typeof TabsPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger>
>(({ children, value, className, ...props }, ref) => {
  return (
    <TabsPrimitive.Trigger
      ref={ref}
      className={cn(
        'text-basis border-subtle bg-canvasBase hover:bg-canvasMuted hover:border-muted data-[state=active]:bg-canvasSubtle data-[state=active]:border-contrast data-[state=active]:border-1 rounded-sm border px-2 py-1.5 text-sm',
        className
      )}
      value={value}
      {...props}
    >
      {children}
    </TabsPrimitive.Trigger>
  );
});

const TabList = forwardRef<
  React.ElementRef<typeof TabsPrimitive.List>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.List>
>(({ children, ...props }, ref) => {
  return (
    <TabsPrimitive.List ref={ref} className="mb-4 flex items-center gap-1.5" {...props}>
      {children}
    </TabsPrimitive.List>
  );
});

const TabContent = forwardRef<
  React.ElementRef<typeof TabsPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Content>
>(({ children, className, value, ...props }, ref) => {
  return (
    <TabsPrimitive.Content value={value} asChild {...props} ref={ref}>
      <div className={cn('text-basis border-subtle rounded-md border px-6 py-4', className)}>
        {children}
      </div>
    </TabsPrimitive.Content>
  );
});

TabCards.Button = TabButton;
TabCards.ButtonList = TabList;
TabCards.Content = TabContent;
