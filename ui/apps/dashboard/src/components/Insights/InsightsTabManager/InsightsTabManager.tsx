'use client';

import { useCallback, useMemo, useState } from 'react';
import { ulid } from 'ulid';

import { InsightsStateMachineContextProvider } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { Query, QuerySnapshot, QueryTemplate } from '@/components/Insights/types';
import { isQuerySnapshot, isQueryTemplate } from '../queries';
import { InsightsTabPanel } from './InsightsTabPanel';
import { InsightsTabsList } from './InsightsTabsList';
import { HOME_TAB, TEMPLATES_TAB, UNTITLED_QUERY } from './constants';

export interface TabManagerActions {
  breakQueryAssociation: (id: string) => void;
  closeTab: (id: string) => void;
  createNewTab: () => void;
  createTabFromQuery: (query: Query | QuerySnapshot | QueryTemplate) => void;
  focusTab: (id: string) => void;
  openTemplatesTab: () => void;
  updateTab: (id: string, patch: Partial<Omit<Query, 'id'>>) => void;
}

export interface UseInsightsTabManagerReturn {
  actions: TabManagerActions;
  activeTabId: string;
  tabManager: JSX.Element;
  tabs: Query[];
}

export interface UseInsightsTabManagerProps {
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
}

export function useInsightsTabManager(
  props: UseInsightsTabManagerProps
): UseInsightsTabManagerReturn {
  const [tabs, setTabs] = useState<Query[]>([HOME_TAB]);
  const [activeTabId, setActiveTabId] = useState<string>(HOME_TAB.id);

  const createTabBase = useCallback(
    (query: Query) => {
      setTabs((prev) => [...prev, query]);
      setActiveTabId(query.id);
    },
    [setActiveTabId]
  );

  const actions = useMemo(
    () => ({
      breakQueryAssociation: (id: string) => {
        const isOpen = activeTabId === id;
        const replacementId = ulid();

        setTabs((prevTabs) =>
          prevTabs.map((tab) => (tab.id === id ? { ...tab, id: replacementId, saved: false } : tab))
        );

        if (isOpen) setActiveTabId(replacementId);
      },
      closeTab: (id: string) => {
        setTabs((prevTabs) => {
          const newTabs = prevTabs.filter((tab) => tab.id !== id);

          const newActiveTabId = getNewActiveTabAfterClose(prevTabs, id, activeTabId);
          if (newActiveTabId !== undefined) setActiveTabId(newActiveTabId);

          return newTabs;
        });
      },
      createNewTab: () => {
        createTabBase(makeEmptyUnsavedQuery());
      },
      createTabFromQuery: (query: Query | QuerySnapshot | QueryTemplate) => {
        if (isQueryTemplate(query)) {
          createTabBase({ ...makeEmptyUnsavedQuery(), query: query.query, name: query.name });
          return;
        }

        if (isQuerySnapshot(query)) {
          createTabBase({ ...makeEmptyUnsavedQuery(), query: query.query });
          return;
        }

        const tabWithSameSavedQueryId = findTabWithId(query.id, tabs);
        if (tabWithSameSavedQueryId !== undefined) {
          setActiveTabId(tabWithSameSavedQueryId.id);
          return;
        }

        createTabBase({
          ...query,
          id: query.saved ? query.id : ulid(),
          name: query.saved ? query.name : UNTITLED_QUERY,
        });
      },
      focusTab: setActiveTabId,
      openTemplatesTab: () => {
        const existingTab = findTabWithId(TEMPLATES_TAB.id, tabs);
        if (existingTab === undefined) {
          createTabBase(TEMPLATES_TAB);
        } else {
          setActiveTabId(TEMPLATES_TAB.id);
        }
      },
      updateTab: (id: string, tab: Partial<Omit<Query, 'id'>>) => {
        setTabs((prevTabs) => prevTabs.map((t) => (t.id === id ? { ...t, ...tab } : t)));
      },
    }),
    [activeTabId, createTabBase, tabs]
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
  tabs: Query[];
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
            onQueryChange={(query) => actions.updateTab(tab.id, { query })}
            onQueryNameChange={(name) => actions.updateTab(tab.id, { name })}
            query={tab.query}
            queryName={tab.name}
            renderChildren={tab.id === activeTabId}
            tabId={tab.id}
          >
            <InsightsTabPanel
              isHomeTab={tab.id === HOME_TAB.id}
              isTemplatesTab={tab.id === TEMPLATES_TAB.id}
              tab={tab}
            />
          </InsightsStateMachineContextProvider>
        ))}
      </div>
    </div>
  );
}

function getNewActiveTabAfterClose(
  existingTabs: Query[],
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

function findTabWithId(id: string, tabs: Query[]): undefined | Query {
  return tabs.find((tab) => tab.id === id);
}

export function hasDiffWithSavedQuery(savedQueries: Record<string, Query>, tab: Query): boolean {
  const savedQuery = savedQueries[tab.id];
  if (savedQuery === undefined) return false;

  return savedQuery.name !== tab.name || savedQuery.query !== tab.query;
}

function makeEmptyUnsavedQuery(): Query {
  return { id: ulid(), name: UNTITLED_QUERY, query: '', saved: false };
}
