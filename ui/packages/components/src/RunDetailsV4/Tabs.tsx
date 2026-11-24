import { useEffect, useState, type ReactNode } from 'react';
import * as TabsPrimitive from '@radix-ui/react-tabs';

export type TabType = {
  label: ReactNode;
  id: string;
  node: ReactNode;
};

export type TabsType = TabType[];

export const Tabs = ({
  tabs,
  defaultActive = '',
  label,
}: {
  tabs: TabsType;
  defaultActive?: string;
  label: string;
}) => {
  const [active, setActive] = useState(defaultActive);

  useEffect(() => {
    setActive(defaultActive);
  }, [defaultActive]);

  const value = tabs.some((tab) => tab.id === active) ? active : (tabs[0]?.id ?? '');

  return (
    <TabsPrimitive.Root
      className="flex h-full w-full flex-col"
      onValueChange={setActive}
      value={value}
    >
      <TabsPrimitive.List
        aria-label={label}
        className="border-muted flex w-full shrink-0 flex-row gap-4 border-b px-4"
      >
        {tabs.map((tab) => (
          <TabsPrimitive.Trigger
            key={tab.id}
            className="text-muted focus-visible:ring-primary-moderate data-[state=active]:text-basis data-[state=active]:border-contrast flex h-[30px] cursor-pointer items-center self-center border-b-2 border-transparent text-sm leading-tight outline-none focus-visible:ring-2 focus-visible:ring-inset"
            value={tab.id}
          >
            {tab.label}
          </TabsPrimitive.Trigger>
        ))}
      </TabsPrimitive.List>
      <div className="relative min-h-0 flex-1 overflow-y-auto">
        {tabs.map((tab) => (
          <TabsPrimitive.Content key={tab.id} className="h-full w-full" value={tab.id}>
            {tab.id === value && tab.node}
          </TabsPrimitive.Content>
        ))}
      </div>
    </TabsPrimitive.Root>
  );
};
