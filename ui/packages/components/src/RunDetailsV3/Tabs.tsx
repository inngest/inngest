import { useState, type ReactNode } from 'react';

import { Timeline } from './Timeline';
import { Workflow } from './Workflow';

export const Tab = ({
  active,
  children,
  onClick,
}: {
  active: boolean;
  children: ReactNode;
  onClick: () => void;
}) => {
  return (
    <div
      className={`${
        active ? 'text-basis border-contrast ' : 'text-muted border-transparent'
      } flex h-[30px] cursor-pointer items-center self-center border-b-2 text-sm leading-tight outline-none`}
      onClick={onClick}
    >
      {children}
    </div>
  );
};

export type TabType = {
  label: string;
  node: ReactNode;
};

export type TabsType = TabType[];

export const Tabs = ({ tabs, defaultActive = 0 }: { tabs: TabsType; defaultActive?: number }) => {
  const [active, setActive] = useState(defaultActive);

  return (
    <div className="flex w-full flex-col">
      <div className="border-muted flex w-full flex-row gap-4 border-b px-4">
        {tabs.map((t: TabType, i: number) => (
          <Tab key={`tab-${i}`} active={active === i} onClick={() => setActive(i)}>
            {t.label}
          </Tab>
        ))}
      </div>
      <div className="relative">
        {tabs.map((tab, i) => (
          <div
            key={`content-${i}`}
            className={`w-full transition-all duration-200 ${
              active === i ? 'visible opacity-100' : 'invisible h-0 opacity-0'
            }`}
          >
            {tab.node}
          </div>
        ))}
      </div>
    </div>
  );
};
