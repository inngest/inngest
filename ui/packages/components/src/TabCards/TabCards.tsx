import { cn } from '@inngest/components/utils/classNames';
import * as TabsPrimitive from '@radix-ui/react-tabs';

import { Card } from '../Card';

export default function TabCards({
  children,
  ...props
}: React.ComponentPropsWithoutRef<typeof TabsPrimitive.Root>) {
  return <TabsPrimitive.Root {...props}>{children}</TabsPrimitive.Root>;
}

function TabButton({
  children,
  value,
  className,
  ...props
}: React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger>) {
  return (
    <TabsPrimitive.Trigger
      className={cn(
        'border-subtle bg-canvasBase hover:bg-canvasMuted hover:border-muted data-[state=active]:bg-canvasSubtle data-[state=active]:border-contrast data-[state=active]:border-1 rounded-sm border px-2 py-1.5 text-sm',
        className
      )}
      value={value}
      {...props}
    >
      {children}
    </TabsPrimitive.Trigger>
  );
}

function TabList({
  children,
  ...props
}: React.ComponentPropsWithoutRef<typeof TabsPrimitive.TabsList>) {
  return (
    <TabsPrimitive.List className="mb-4 flex items-center gap-1.5" {...props}>
      {children}
    </TabsPrimitive.List>
  );
}

function TabContent({
  children,
  value,
  ...props
}: React.ComponentPropsWithoutRef<typeof TabsPrimitive.TabsContent>) {
  return (
    <TabsPrimitive.Content value={value} asChild {...props}>
      <Card>
        <Card.Content>{children}</Card.Content>
      </Card>
    </TabsPrimitive.Content>
  );
}

TabCards.Button = TabButton;
TabCards.ButtonList = TabList;
TabCards.Content = TabContent;
