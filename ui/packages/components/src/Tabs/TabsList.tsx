import { cn } from '@inngest/components/utils/classNames';
import * as TabsPrimitive from '@radix-ui/react-tabs';

type TabsListProps = React.ComponentPropsWithoutRef<typeof TabsPrimitive.List>;

export function TabsList({ children, className, ...props }: TabsListProps) {
  return (
    <TabsPrimitive.List
      className={cn(
        'border-subtle border-b',
        'flex',
        'overflow-x-auto [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden',
        className
      )}
      {...props}
    >
      {children}
    </TabsPrimitive.List>
  );
}
