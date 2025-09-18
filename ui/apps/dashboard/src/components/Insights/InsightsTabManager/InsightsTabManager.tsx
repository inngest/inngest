'use client';

import { useCallback, useMemo, useState } from 'react';
import { ulid } from 'ulid';

import { InsightsStateMachineContextProvider } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { QuerySnapshot, QueryTemplate, Tab } from '@/components/Insights/types';
import type { InsightsQuery } from '@/gql/graphql';
import { isQuerySnapshot, isQueryTemplate } from '../queries';
import { InsightsTabPanel } from './InsightsTabPanel';
import { InsightsTabsList } from './InsightsTabsList';
import { HOME_TAB, TEMPLATES_TAB, UNTITLED_QUERY } from './constants';

export interface TabManagerActions {
  breakQueryAssociation: (savedQueryId: string) => void;
  closeTab: (id: string) => void;
  createNewTab: () => void;
  createTabFromQuery: (query: InsightsQuery | QuerySnapshot | QueryTemplate) => void;
  focusTab: (id: string) => void;
  openTemplatesTab: () => void;
  updateTab: (id: string, patch: Partial<Omit<Tab, 'id'>>) => void;
}

export interface UseInsightsTabManagerReturn {
  actions: TabManagerActions;
  activeTabId: string;
  tabManager: JSX.Element;
  tabs: Tab[];
}

export interface UseInsightsTabManagerProps {
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
}

export function useInsightsTabManager(
  props: UseInsightsTabManagerProps
): UseInsightsTabManagerReturn {
  const [tabs, setTabs] = useState<Tab[]>([HOME_TAB]);
  const [activeTabId, setActiveTabId] = useState<string>(HOME_TAB.id);

  const createTabBase = useCallback(
    (tab: Tab) => {
      setTabs((prev) => [...prev, tab]);
      setActiveTabId(tab.id);
    },
    [setActiveTabId]
  );

  const actions = useMemo(
    () => ({
      breakQueryAssociation: (savedQueryId: string) => {
        setTabs((prevTabs) =>
          prevTabs.map((tab) =>
            tab.savedQueryId === savedQueryId ? { ...tab, savedQueryId: undefined } : tab
          )
        );
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
        createTabBase(makeEmptyUnsavedTab());
      },
      createTabFromQuery: (query: InsightsQuery | QuerySnapshot | QueryTemplate) => {
        if (isQueryTemplate(query)) {
          createTabBase({ ...makeEmptyUnsavedTab(), query: query.query, name: query.name });
          return;
        }

        if (isQuerySnapshot(query)) {
          createTabBase({ ...makeEmptyUnsavedTab(), query: query.query });
          return;
        }

        const tabWithSameSavedQueryId = tabs.find((tab) => tab.savedQueryId === query.id);
        if (tabWithSameSavedQueryId !== undefined) {
          setActiveTabId(tabWithSameSavedQueryId.id);
          return;
        }

        createTabBase({ id: ulid(), name: query.name, query: query.sql, savedQueryId: query.id });
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
      updateTab: (id: string, tab: Partial<Omit<Tab, 'id'>>) => {
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
  tabs: Tab[];
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
  existingTabs: Tab[],
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

function findTabWithId(id: string, tabs: Tab[]): undefined | Tab {
  return tabs.find((tab) => tab.id === id);
}

export function hasDiffWithSavedQuery(
  savedQueries: InsightsQuery[] | undefined,
  tab: Tab
): boolean {
  if (tab.savedQueryId === undefined || savedQueries === undefined) return false;
  const savedQuery = savedQueries.find((q) => q.id === tab.savedQueryId);
  if (!savedQuery) return false;
  return savedQuery.name !== tab.name || savedQuery.sql !== tab.query;
}

/**
 * Determines whether the given tab represents a saved query.
 *
 * Note on slow/failed query list updates:
 *   - We set `tab.savedQueryId` immediately after a successful create mutation, so
 *   the tab is considered "saved" on the very next render, even before the
 *   saved queries list refetch completes.
 */
export function getIsSavedQuery(tab: Tab): boolean {
  return tab.savedQueryId !== undefined;
}

/**
 * Computes whether the Save/Update button should be disabled for a tab.
 *
 * Behavior in slow/failed query list refetch scenarios:
 * - If the tab was just saved, `tab.savedQueryId` is set immediately.
 * - Until the saved queries list refetches (or if it fails), the corresponding
 *   saved query may not be present in `savedQueries`.
 * - `hasDiffWithSavedQuery` returns false when the saved query is not found.
 *   Combined with `isSaved`, this keeps the button disabled, preventing a brief
 *   period where the UI would show "Update" but allow clicking while data is
 *   still syncing.
 */
export function getDisableSaveOrUpdate(
  savedQueries: InsightsQuery[] | undefined,
  tab: Tab
): boolean {
  const isSavedQuery = getIsSavedQuery(tab);
  return tab.name === '' || (isSavedQuery && !hasDiffWithSavedQuery(savedQueries, tab));
}

function makeEmptyUnsavedTab(): Tab {
  return { id: ulid(), name: UNTITLED_QUERY, query: '' };
}
