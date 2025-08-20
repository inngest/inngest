'use client';

import { cn } from '@inngest/components/utils/classNames';
import * as TabsPrimitive from '@radix-ui/react-tabs';

import { IconTab } from './IconTab';
import { Tab } from './Tab';
import { TabsContent } from './TabsContent';
import { TabsContext, type TabsContextValue } from './TabsContext';
import { TabsList } from './TabsList';

interface TabsProps extends React.ComponentPropsWithoutRef<typeof TabsPrimitive.Root> {
  defaultIconBefore?: React.ReactNode;
  onClose?: (value: string) => void;
  variant?: TabsContextValue['variant'];
}

function Tabs({
  children,
  className,
  defaultIconBefore,
  onClose,
  variant = 'dynamic',
  ...props
}: TabsProps) {
  return (
    <TabsContext.Provider value={{ defaultIconBefore, onClose, variant }}>
      <TabsPrimitive.Root className={cn('flex w-full flex-col', className)} {...props}>
        {children}
      </TabsPrimitive.Root>
    </TabsContext.Provider>
  );
}

Tabs.Content = TabsContent;
Tabs.IconTab = IconTab;
Tabs.List = TabsList;
Tabs.Tab = Tab;

export default Tabs;
