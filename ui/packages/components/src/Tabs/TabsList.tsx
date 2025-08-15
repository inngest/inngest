import { cn } from '@inngest/components/utils/classNames';
import * as TabsPrimitive from '@radix-ui/react-tabs';

const LIST_BORDER_STYLES = 'border-b border-subtle';
const LIST_LAYOUT_STYLES = 'flex items-center w-full';
const LIST_SCROLL_STYLES =
  'overflow-x-auto [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden';

type TabsListProps = React.ComponentPropsWithoutRef<typeof TabsPrimitive.List>;

export function TabsList({ children, className, ...props }: TabsListProps) {
  return (
    <TabsPrimitive.List
      className={cn(LIST_BORDER_STYLES, LIST_LAYOUT_STYLES, LIST_SCROLL_STYLES, className)}
      {...props}
    >
      {children}
    </TabsPrimitive.List>
  );
}
