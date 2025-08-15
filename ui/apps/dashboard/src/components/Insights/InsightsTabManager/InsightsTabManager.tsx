'use client';

import { useCallback, useMemo, useState } from 'react';
import { ulid } from 'ulid';

import { InsightsStateMachineContextProvider } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { Query } from '../QueryHelperPanel/types';
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
  createTabFromQuery: (query: Query) => void;
  focusTab: (id: string) => void;
  openTemplatesTab: () => void;
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

  const focusTabBase = useCallback(
    (tabId: string, updatedTabs?: TabConfig[]) => {
      const relevantTabs = updatedTabs ?? tabs;

      const tab = relevantTabs.find((tab) => tab.id === tabId);
      if (tab === undefined) {
        console.warn('Attempted to focus a tab that does not exist.');
        return;
      }

      setActiveTabId(tabId);
    },
    [tabs]
  );

  const createTabBase = useCallback(
    (query: Query): TabConfig[] => {
      const savedQueryId = query.isSavedQuery ? query.id : undefined;
      const tabWithSameSavedQueryId =
        savedQueryId !== undefined
          ? tabs.find((tab) => tab.savedQueryId === savedQueryId)
          : undefined;
      if (tabWithSameSavedQueryId !== undefined) {
        focusTabBase(tabWithSameSavedQueryId.id);
        return tabs;
      }

      const updatedTabs = [...tabs, { ...query, savedQueryId }];
      setTabs(updatedTabs);
      focusTabBase(query.id, updatedTabs);

      return updatedTabs;
    },
    [focusTabBase, tabs]
  );

  const actions = useMemo(
    () => ({
      closeTab: (id: string) => {
        setTabs((prevTabs) => {
          if (prevTabs.find((tab) => tab.id === id) === undefined) {
            console.warn('Attempted to close a tab that does not exist.');
            return prevTabs;
          }

          const newTabs = prevTabs.filter((tab) => tab.id !== id);

          const newActiveTabId = getNewActiveTabAfterClose(prevTabs, id, activeTabId);
          if (newActiveTabId !== undefined) {
            focusTabBase(newActiveTabId, newTabs);
          }

          return newTabs;
        });
      },
      createNewTab: () => {
        createTabBase({ id: ulid(), isSavedQuery: false, name: 'Untitled query', query: '' });
      },
      createTabFromQuery: (query: Query) => {
        const id = query.isSavedQuery ? query.id : ulid();
        const name = query.isSavedQuery ? query.name : 'Untitled query';
        createTabBase({ ...query, id, name });
      },
      focusTab: focusTabBase,
      openTemplatesTab: () => {
        const existingTab = tabs.find((tab) => tab.id === TEMPLATES_TAB.id);
        if (existingTab === undefined) {
          const newTabs = createTabBase({ ...TEMPLATES_TAB, isSavedQuery: false });
          focusTabBase(TEMPLATES_TAB.id, newTabs);
        } else {
          focusTabBase(TEMPLATES_TAB.id);
        }
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
            <InsightsTabPanel isTemplatesTab={tab.id === TEMPLATES_TAB.id} />
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
): undefined | string {
  if (tabIdToClose !== currentActiveTabId) return currentActiveTabId;

  const closingTabIndex = existingTabs.findIndex((tab) => tab.id === tabIdToClose);
  if (closingTabIndex === -1) return currentActiveTabId;

  // 1: Try to select the next tab (now where the closed tab was).
  // 2: Try to select the tab before the closed tab.
  const remainingTabs = existingTabs.filter((tab) => tab.id !== tabIdToClose);
  const newlySelectedTabId =
    remainingTabs[closingTabIndex]?.id ?? remainingTabs[closingTabIndex - 1]?.id;
  return newlySelectedTabId;
}
