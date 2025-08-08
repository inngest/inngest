'use client';

import { useMemo, useState } from 'react';

import { InsightsStateMachineContextProvider } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { InsightsTabPanel } from './InsightsTabPanel';
import { InsightsTabsList } from './InsightsTabsList';

const HOME_TAB = {
  id: '__home',
  name: 'Home',
  query: '',
} as const;

export interface TabConfig {
  id: string;
  name: string;
  query: string;
  savedQueryId?: string;
}

export interface TabManagerActions {
  closeTab: (id: string) => void;
  createTab: (id: string, name?: string) => void;
  focusTab: (id: string) => void;
  updateTabQuery: (id: string, query: string) => void;
}

export interface UseInsightsTabManagerReturn {
  actions: TabManagerActions;
  activeTabId: string;
  tabManager: JSX.Element;
  tabs: TabConfig[];
}

export function useInsightsTabManager(): UseInsightsTabManagerReturn {
  const [tabs, setTabs] = useState<TabConfig[]>([HOME_TAB]);
  const [activeTabId, setActiveTabId] = useState<string>(HOME_TAB.id);

  const actions = useMemo(
    () => ({
      closeTab: (id: string) => {
        if (id === HOME_TAB.id) return;

        setTabs((prevTabs) => {
          const tabIndex = prevTabs.findIndex((tab) => tab.id === id);
          if (tabIndex === -1) return prevTabs;

          const newActiveTabId = getNewActiveTabAfterClose(prevTabs, id, activeTabId);
          setActiveTabId(newActiveTabId);

          return prevTabs.filter((tab) => tab.id !== id);
        });
      },
      createTab: (id: string, name = 'Untitled query') => {
        if (tabs.some((tab) => tab.id === id)) return;

        setTabs((prevTabs) => [...prevTabs, { id, name, query: '' }]);
        setActiveTabId(id);
      },
      focusTab: (id: string) => {
        const tab = tabs.find((tab) => tab.id === id);
        if (tab !== undefined) setActiveTabId(id);
      },
      updateTabQuery: (id: string, query: string) => {
        setTabs((prevTabs) => prevTabs.map((tab) => (tab.id === id ? { ...tab, query } : tab)));
      },
    }),
    [activeTabId, tabs]
  );

  const tabManager = useMemo(
    () => <InsightsTabManagerInternal actions={actions} activeTabId={activeTabId} tabs={tabs} />,
    [actions, activeTabId, tabs]
  );

  return { actions, activeTabId, tabManager, tabs };
}

interface InsightsTabManagerInternalProps {
  activeTabId: string;
  tabs: TabConfig[];
  actions: TabManagerActions;
}

function InsightsTabManagerInternal({
  tabs,
  activeTabId,
  actions,
}: InsightsTabManagerInternalProps) {
  return (
    <div className="flex h-full w-full flex-1 flex-col overflow-hidden">
      <InsightsTabsList actions={actions} activeTabId={activeTabId} hide tabs={tabs} />
      <div className="grid h-full w-full flex-1 grid-rows-[3fr_5fr] gap-0 overflow-hidden">
        {tabs.map((tab) => (
          <InsightsStateMachineContextProvider key={tab.id} renderChildren={tab.id === activeTabId}>
            <InsightsTabPanel />
          </InsightsStateMachineContextProvider>
        ))}
      </div>
    </div>
  );
}

function getNewActiveTabAfterClose(
  existingTabs: TabConfig[],
  tabIdToClose: string,
  currentActiveTabId: string
): string {
  if (tabIdToClose !== currentActiveTabId) return currentActiveTabId;

  const closingTabIndex = existingTabs.findIndex((tab) => tab.id === tabIdToClose);
  if (closingTabIndex === -1) return currentActiveTabId;

  // 1: Try to select the next tab (now where the closed tab was).
  // 2: Try to select the tab before the closed tab.
  // 3: Fallback to the home tab.
  const remainingTabs = existingTabs.filter((tab) => tab.id !== tabIdToClose);
  const newlySelectedTabId =
    remainingTabs[closingTabIndex]?.id ?? remainingTabs[closingTabIndex - 1]?.id ?? HOME_TAB.id;
  return newlySelectedTabId;
}
