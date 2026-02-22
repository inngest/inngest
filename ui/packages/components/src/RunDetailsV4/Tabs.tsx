import { useEffect, useState, type ReactNode } from 'react';

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
        active ? 'text-basis border-contrast' : 'text-muted border-transparent'
      } flex h-[30px] cursor-pointer items-center self-center border-b-2 text-sm leading-tight outline-none`}
      onClick={onClick}
    >
      {children}
    </div>
  );
};

export type TabType = {
  label: string;
  id: string;
  node: ReactNode;
};

export type TabsType = TabType[];

export const Tabs = ({ tabs, defaultActive = '' }: { tabs: TabsType; defaultActive?: string }) => {
  const [active, setActive] = useState(defaultActive);

  useEffect(() => {
    setActive(defaultActive);
  }, [defaultActive]);

  return (
    <div className="flex h-full w-full flex-col">
      <div className="border-muted flex w-full shrink-0 flex-row gap-4 border-b px-4">
        {tabs.map((t: TabType, i: number) => (
          <Tab
            key={`tab-${i}`}
            active={active === t.id || (active === '' && i === 0)}
            onClick={() => setActive(t.id)}
          >
            {t.label}
          </Tab>
        ))}
      </div>
      <div className="relative min-h-0 flex-1 overflow-y-auto">
        {tabs.map((tab, i) => {
          const tabActive = active === tab.id || (active === '' && i === 0);
          return (
            <div
              key={`content-${i}`}
              className={`w-full ${
                tabActive ? 'visible h-full opacity-100' : 'invisible h-0 opacity-0'
              }`}
            >
              {tabActive && tab.node}
            </div>
          );
        })}
      </div>
    </div>
  );
};
