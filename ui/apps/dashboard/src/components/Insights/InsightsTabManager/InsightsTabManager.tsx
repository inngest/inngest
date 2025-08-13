'use client';

import { useMemo, useState } from 'react';
import { ulid } from 'ulid';

import { InsightsStateMachineContextProvider } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { Query } from '../QueryHelperPanel';
import { InsightsTabPanel } from './InsightsTabPanel';
import { InsightsTabsList } from './InsightsTabsList';
import { TEMPLATES_TAB } from './constants';

export interface TabConfig {
  id: string;
  name: string;
  query: string;
  savedQueryId?: string;
}

export interface TabManagerActions {
  closeTab: (id: string) => void;
  createNewTab: () => void;
  createTab: (query: Query) => void;
  focusOrCreateTemplatesTab: () => void;
  focusTab: (id: string) => void;
  getTabIdForSavedQuery: (savedQueryId: string) => undefined | string;
  updateTabQuery: (id: string, query: string) => void;
}

export interface UseInsightsTabManagerReturn {
  actions: TabManagerActions;
  activeTabId: string;
  tabManager: JSX.Element;
  tabs: TabConfig[];
}

export interface UseInsightsTabManagerProps {
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
}

export function useInsightsTabManager(
  props: UseInsightsTabManagerProps
): UseInsightsTabManagerReturn {
  const [tabs, setTabs] = useState<TabConfig[]>([TEMPLATES_TAB]);
  const [activeTabId, setActiveTabId] = useState<string>(TEMPLATES_TAB.id);

  const actions = useMemo(
    () => ({
      closeTab: (id: string) => {
        setTabs((prevTabs) => {
          const tabIndex = prevTabs.findIndex((tab) => tab.id === id);
          if (tabIndex === -1) return prevTabs;

          const newActiveTabId = getNewActiveTabAfterClose(prevTabs, id, activeTabId);
          setActiveTabId(newActiveTabId);

          return prevTabs.filter((tab) => tab.id !== id);
        });
      },
      createNewTab: () => {
        const newTabId = ulid();

        setTabs((prevTabs) => [
          ...prevTabs,
          {
            id: newTabId,
            name: 'Untitled query',
            query: '',
          },
        ]);

        setActiveTabId(newTabId);
      },
      createTab: (query: Query) => {
        if (tabs.some((tab) => tab.savedQueryId === query.id)) return;

        const newTabId = ulid();

        setTabs((prevTabs) => [
          ...prevTabs,
          {
            id: newTabId,
            name: query.name,
            query: query.query,
            savedQueryId: query.type === 'saved' ? query.id : undefined,
          },
        ]);

        setActiveTabId(newTabId);
      },
      focusOrCreateTemplatesTab: () => {
        const existingTab = tabs.find((tab) => tab.id === TEMPLATES_TAB.id);
        if (existingTab) {
          setActiveTabId(TEMPLATES_TAB.id);
        } else {
          setTabs((prevTabs) => [TEMPLATES_TAB, ...prevTabs]);
          setActiveTabId(TEMPLATES_TAB.id);
        }
      },
      focusTab: (id: string) => {
        const tab = tabs.find((tab) => tab.id === id);
        if (tab !== undefined) setActiveTabId(id);
      },
      getTabIdForSavedQuery: (savedQueryId: string) => {
        return tabs.find((tab) => tab.savedQueryId === savedQueryId)?.id;
      },
      updateTabQuery: (id: string, query: string) => {
        setTabs((prevTabs) => prevTabs.map((tab) => (tab.id === id ? { ...tab, query } : tab)));
      },
    }),
    [activeTabId, tabs]
  );

  const tabManager = useMemo(
    () => (
      <InsightsTabManagerInternal
        actions={actions}
        activeTabId={activeTabId}
        tabs={tabs}
        isQueryHelperPanelVisible={props.isQueryHelperPanelVisible}
        onToggleQueryHelperPanelVisibility={props.onToggleQueryHelperPanelVisibility}
      />
    ),
    [
      actions,
      activeTabId,
      tabs,
      props.isQueryHelperPanelVisible,
      props.onToggleQueryHelperPanelVisibility,
    ]
  );

  return { actions, activeTabId, tabManager, tabs };
}

interface InsightsTabManagerInternalProps {
  actions: TabManagerActions;
  activeTabId: string;
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
  tabs: TabConfig[];
}

function InsightsTabManagerInternal({
  tabs,
  activeTabId,
  actions,
  isQueryHelperPanelVisible,
  onToggleQueryHelperPanelVisibility,
}: InsightsTabManagerInternalProps) {
  return (
    <div className="flex h-full w-full flex-1 flex-col overflow-hidden">
      <InsightsTabsList
        actions={actions}
        activeTabId={activeTabId}
        isQueryHelperPanelVisible={isQueryHelperPanelVisible}
        onToggleQueryHelperPanelVisibility={onToggleQueryHelperPanelVisibility}
        tabs={tabs}
      />
      <div className="grid h-full w-full flex-1 grid-rows-[3fr_5fr] gap-0 overflow-hidden">
        {tabs.map((tab) => (
          <InsightsStateMachineContextProvider
            key={tab.id}
            onQueryChange={(query) => actions.updateTabQuery(tab.id, query)}
            query={tab.query}
            renderChildren={tab.id === activeTabId}
          >
            <InsightsTabPanel
              isTemplatesTab={tab.id === TEMPLATES_TAB.id}
              tabManagerActions={actions}
            />
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
  // 3: Fallback to empty string if no tabs remain.
  const remainingTabs = existingTabs.filter((tab) => tab.id !== tabIdToClose);
  const newlySelectedTabId =
    remainingTabs[closingTabIndex]?.id ?? remainingTabs[closingTabIndex - 1]?.id ?? '';
  return newlySelectedTabId;
}
