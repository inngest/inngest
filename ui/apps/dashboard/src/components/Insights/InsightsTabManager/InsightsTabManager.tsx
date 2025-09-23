'use client';

import { useCallback, useMemo, useRef, useState, type ReactNode } from 'react';
import { useUser } from '@clerk/nextjs';
import { AgentProvider, createInMemorySessionTransport } from '@inngest/use-agents';
import { ulid } from 'ulid';
import { v4 as uuidv4 } from 'uuid';

import { InsightsStateMachineContextProvider } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { QuerySnapshot, QueryTemplate, Tab } from '@/components/Insights/types';
import type { InsightsQueryStatement } from '@/gql/graphql';
import { InsightsChat } from '../InsightsChat/InsightsChat';
import { isQuerySnapshot, isQueryTemplate } from '../queries';
import { InsightsTabPanel } from './InsightsTabPanel';
import { InsightsTabsList } from './InsightsTabsList';
import { HOME_TAB, TEMPLATES_TAB, UNTITLED_QUERY } from './constants';

export interface TabManagerActions {
  breakQueryAssociation: (savedQueryId: string) => void;
  closeTab: (id: string) => void;
  createNewTab: () => void;
  createTabFromQuery: (query: InsightsQueryStatement | QuerySnapshot | QueryTemplate) => void;
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
  historyWindow?: number;
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
}

export function useInsightsTabManager(
  props: UseInsightsTabManagerProps
): UseInsightsTabManagerReturn {
  const [tabs, setTabs] = useState<Tab[]>([HOME_TAB]);
  const [activeTabId, setActiveTabId] = useState<string>(HOME_TAB.id);
  const [isChatPanelVisible, setIsChatPanelVisible] = useState(true);

  const onToggleChatPanelVisibility = useCallback(() => {
    setIsChatPanelVisible((prev) => !prev);
  }, []);

  const threadIdMapRef = useRef<Record<string, string>>({});
  const getThreadIdForTab = useCallback((tabId: string): string => {
    const existing = threadIdMapRef.current[tabId];
    if (existing) return existing;
    const id = uuidv4();
    threadIdMapRef.current[tabId] = id;
    return id;
  }, []);

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
      createTabFromQuery: (query: InsightsQueryStatement | QuerySnapshot | QueryTemplate) => {
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

        createTabBase({
          id: ulid(),
          name: query.name,
          query: query.sql,
          savedQueryId: query.id,
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
        getThreadIdForTab={getThreadIdForTab}
        historyWindow={props.historyWindow}
        isQueryHelperPanelVisible={props.isQueryHelperPanelVisible}
        onToggleQueryHelperPanelVisibility={props.onToggleQueryHelperPanelVisibility}
        isChatPanelVisible={isChatPanelVisible}
        onToggleChatPanelVisibility={onToggleChatPanelVisibility}
      />
    ),
    [
      actions,
      activeTabId,
      tabs,
      getThreadIdForTab,
      props.historyWindow,
      props.isQueryHelperPanelVisible,
      props.onToggleQueryHelperPanelVisibility,
      isChatPanelVisible,
      onToggleChatPanelVisibility,
    ]
  );

  return { actions, activeTabId, tabManager, tabs };
}

interface InsightsTabManagerInternalProps {
  actions: TabManagerActions;
  activeTabId: string;
  getThreadIdForTab: (tabId: string) => string;
  historyWindow?: number;
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
  tabs: Tab[];
  isChatPanelVisible: boolean;
  onToggleChatPanelVisibility: () => void;
}

function InsightsTabManagerInternal({
  tabs,
  activeTabId,
  actions,
  getThreadIdForTab,
  historyWindow,
  isQueryHelperPanelVisible,
  onToggleQueryHelperPanelVisibility,
  isChatPanelVisible,
  onToggleChatPanelVisibility,
}: InsightsTabManagerInternalProps) {
  // Provide shared transport/connection for all descendant useAgents hooks
  const { user } = useUser();
  const transport = useMemo(() => createInMemorySessionTransport(), []);
  const channelKey = user?.id ? `insights:${user.id}` : undefined;
  // Type shim to avoid cross-package ReactNode incompatibilities during local linking
  const AnyAgentProvider = AgentProvider as unknown as React.FC<any>;
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
          <div
            className={
              tab.id === activeTabId ? 'flex h-full w-full' : 'invisible h-0 w-full overflow-hidden'
            }
          >
            <div className="w-full overflow-hidden">
              <InsightsTabPanel
                isHomeTab={tab.id === HOME_TAB.id}
                isTemplatesTab={tab.id === TEMPLATES_TAB.id}
                tab={tab}
                historyWindow={historyWindow}
                isChatPanelVisible={isChatPanelVisible}
                onToggleChatPanelVisibility={onToggleChatPanelVisibility}
              />
            </div>
            {tab.id !== HOME_TAB.id && tab.id !== TEMPLATES_TAB.id && (
              <InsightsChat
                className={!isChatPanelVisible ? 'hidden' : ''}
                threadId={getThreadIdForTab(tab.id)}
                onToggleChat={onToggleChatPanelVisibility}
              />
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

function findTabWithId(id: string, tabs: Tab[]): string | undefined {
  return tabs.find((tab) => tab.id === id)?.id;
}

export function hasDiffWithSavedQuery(
  savedQueries: InsightsQueryStatement[] | undefined,
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

function makeEmptyUnsavedTab(): Tab {
  return { id: ulid(), name: UNTITLED_QUERY, query: '' };
}
