'use client';

import { useCallback, useMemo, useRef, useState, type ReactNode } from 'react';
import { useUser } from '@clerk/nextjs';
import { AgentProvider, createInMemorySessionTransport } from '@inngest/use-agents';
import { ulid } from 'ulid';
import { v4 as uuidv4 } from 'uuid';

import { InsightsStateMachineContextProvider } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { Query, QuerySnapshot, QueryTemplate } from '@/components/Insights/types';
import { InsightsChat } from '../InsightsChat/InsightsChat';
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
  // Provide shared transport/connection for all descendant useAgents hooks
  const { user } = useUser();
  const transport = useMemo(() => createInMemorySessionTransport(), []);
  const channelKey = user?.id ? `insights:${user.id}` : undefined;
  // Type shim to avoid cross-package ReactNode incompatibilities during local linking
  const AnyAgentProvider = AgentProvider as unknown as React.FC<any>;
  // Stable per-tab thread UUID mapping to satisfy server-side UUID validation
  const threadIdMapRef = useRef<Record<string, string>>({});
  const getThreadIdForTab = useCallback((tabId: string): string => {
    const existing = threadIdMapRef.current[tabId];
    if (existing) return existing;
    const id = uuidv4();
    threadIdMapRef.current[tabId] = id;
    return id;
  }, []);
  const providerChildren: ReactNode = (
    <div>
      {tabs.map((tab) => (
        <InsightsStateMachineContextProvider
          key={tab.id}
          onQueryChange={(query) => actions.updateTab(tab.id, { query })}
          onQueryNameChange={(name) => actions.updateTab(tab.id, { name })}
          query={tab.query}
          queryName={tab.name}
          renderChildren={true}
          tabId={tab.id}
        >
          <div className={tab.id === activeTabId ? 'flex h-full w-full' : 'hidden h-full w-full'}>
            <div className="flex-1 overflow-hidden">
              <InsightsTabPanel
                isHomeTab={tab.id === HOME_TAB.id}
                isTemplatesTab={tab.id === TEMPLATES_TAB.id}
                tab={tab}
              />
            </div>
            {tab.id !== HOME_TAB.id && tab.id !== TEMPLATES_TAB.id && (
              <InsightsChat threadId={getThreadIdForTab(tab.id)} />
            )}
          </div>
        </InsightsStateMachineContextProvider>
      ))}
    </div>
  );
  return (
    <div className="flex h-full w-full flex-1 flex-col overflow-hidden">
      <InsightsTabsList
        activeTabId={activeTabId}
        isQueryHelperPanelVisible={isQueryHelperPanelVisible}
        onToggleQueryHelperPanelVisibility={onToggleQueryHelperPanelVisibility}
        tabs={tabs}
      />
      <div className="flex h-full w-full flex-1 overflow-hidden">
        <AnyAgentProvider
          userId={user?.id || undefined}
          channelKey={channelKey}
          transport={transport}
          debug={false}
        >
          {providerChildren}
        </AnyAgentProvider>
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
