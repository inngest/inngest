'use client';

import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { useUser } from '@clerk/nextjs';
import { Resizable } from '@inngest/components/Resizable/Resizable';
import { AgentProvider, createInMemorySessionTransport } from '@inngest/use-agent';
import { RiBookOpenLine, RiFeedbackLine, RiSparkling2Line, RiTable2 } from '@remixicon/react';
import { ulid } from 'ulid';
import { v4 as uuidv4 } from 'uuid';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { InsightsStateMachineContextProvider } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { QuerySnapshot, QueryTemplate, Tab } from '@/components/Insights/types';
import type { InsightsQueryStatement } from '@/gql/graphql';
import {
  InsightsChatProvider,
  useInsightsChatProvider,
} from '../InsightsChat/InsightsChatProvider';
import { isQuerySnapshot, isQueryTemplate } from '../queries';
import { InsightsHelperPanel } from './InsightsHelperPanel';
import { InsightsHelperPanelControl, type HelperItem } from './InsightsHelperPanelControl';
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
  const [isHelperPanelOpen, setIsHelperPanelOpen] = useState(false);
  const isInsightsAgentEnabled = useBooleanFlag('insights-agent');

  const onToggleChatPanelVisibility = useCallback(() => {
    if (!isInsightsAgentEnabled.value) return;
    setIsChatPanelVisible((prev) => !prev);
  }, [isInsightsAgentEnabled.value]);

  const effectiveChatPanelVisible = isInsightsAgentEnabled.value && isChatPanelVisible;

  // Map each UI tab to a stable agent thread id
  const agentThreadIdByTabRef = useRef<Record<string, string>>({});
  const getAgentThreadIdForTab = useCallback((tabId: string): string => {
    const existing = agentThreadIdByTabRef.current[tabId];
    if (existing) return existing;
    const id = uuidv4();
    agentThreadIdByTabRef.current[tabId] = id;
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
        const hasTemplatesTab = tabs.some((tab) => tab.id === TEMPLATES_TAB.id);
        if (!hasTemplatesTab) {
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
        getAgentThreadIdForTab={getAgentThreadIdForTab}
        historyWindow={props.historyWindow}
        isQueryHelperPanelVisible={props.isQueryHelperPanelVisible}
        onToggleQueryHelperPanelVisibility={props.onToggleQueryHelperPanelVisibility}
        isHelperPanelOpen={isHelperPanelOpen}
        setIsHelperPanelOpen={setIsHelperPanelOpen}
        isInsightsAgentEnabled={isInsightsAgentEnabled.value}
      />
    ),
    [
      actions,
      activeTabId,
      tabs,
      getAgentThreadIdForTab,
      props.historyWindow,
      props.isQueryHelperPanelVisible,
      props.onToggleQueryHelperPanelVisibility,
      isHelperPanelOpen,
      setIsHelperPanelOpen,
      isInsightsAgentEnabled.value,
    ]
  );

  return { actions, activeTabId, tabManager, tabs };
}

interface InsightsTabManagerInternalProps {
  actions: TabManagerActions;
  activeTabId: string;
  getAgentThreadIdForTab: (tabId: string) => string;
  historyWindow?: number;
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
  tabs: Tab[];
  isHelperPanelOpen: boolean;
  setIsHelperPanelOpen: (open: boolean) => void;
  isInsightsAgentEnabled: boolean;
}

function InsightsTabManagerInternal({
  tabs,
  activeTabId,
  actions,
  getAgentThreadIdForTab,
  historyWindow,
  isQueryHelperPanelVisible,
  onToggleQueryHelperPanelVisibility,
  isHelperPanelOpen,
  setIsHelperPanelOpen,
  isInsightsAgentEnabled,
}: InsightsTabManagerInternalProps) {
  const [activeHelper, setActiveHelper] = useState<string | null>(null);

  const handleSelectHelper = useCallback(
    (title: string) => {
      if (activeHelper === title && isHelperPanelOpen) {
        setIsHelperPanelOpen(false);
        setActiveHelper(null);
      } else {
        setActiveHelper(title);
        if (!isHelperPanelOpen) setIsHelperPanelOpen(true);
      }
    },
    [activeHelper, isHelperPanelOpen, setIsHelperPanelOpen]
  );

  const helperItems = useMemo<HelperItem[]>(
    () => [
      { title: 'AI', icon: <RiSparkling2Line size={20} />, action: () => handleSelectHelper('AI') },
      {
        title: 'Docs',
        icon: <RiBookOpenLine size={20} />,
        action: () => handleSelectHelper('Docs'),
      },
      {
        title: 'Schemas',
        icon: <RiTable2 size={20} />,
        action: () => handleSelectHelper('Schemas'),
      },
      {
        title: 'Support',
        icon: <RiFeedbackLine size={20} />,
        action: () => handleSelectHelper('Support'),
      },
    ],
    [handleSelectHelper]
  );
  // Provide shared transport/connection for all descendant useAgents hooks
  const { user } = useUser();
  const transport = useMemo(
    () => (isInsightsAgentEnabled ? createInMemorySessionTransport() : undefined),
    [isInsightsAgentEnabled]
  );

  const providerChildren: ReactNode = (
    <div className="h-full w-full">
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
          <div className={tab.id === activeTabId ? 'h-full w-full' : 'h-0 w-full overflow-hidden'}>
            {isHelperPanelOpen ? (
              <div className="flex h-full w-full">
                <div className="min-w-0 flex-1 overflow-hidden">
                  <Resizable
                    defaultSplitPercentage={75}
                    minSplitPercentage={20}
                    maxSplitPercentage={85}
                    orientation="horizontal"
                    splitKey="insights-helper-split"
                    first={
                      <div className="h-full min-w-0 overflow-hidden">
                        <InsightsTabPanel
                          isHomeTab={tab.id === HOME_TAB.id}
                          isTemplatesTab={tab.id === TEMPLATES_TAB.id}
                          tab={tab}
                          historyWindow={historyWindow}
                        />
                      </div>
                    }
                    second={<InsightsHelperPanel active={activeHelper} />}
                  />
                </div>
                {isQueryTab(tab.id) ? (
                  <InsightsHelperPanelControl items={helperItems} activeTitle={activeHelper} />
                ) : null}
              </div>
            ) : (
              <div className="flex h-full w-full">
                <div className="h-full min-w-0 flex-1 overflow-hidden">
                  <InsightsTabPanel
                    isHomeTab={tab.id === HOME_TAB.id}
                    isTemplatesTab={tab.id === TEMPLATES_TAB.id}
                    tab={tab}
                    historyWindow={historyWindow}
                  />
                </div>
                {isQueryTab(tab.id) ? (
                  <InsightsHelperPanelControl items={helperItems} activeTitle={activeHelper} />
                ) : null}
              </div>
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
        {isInsightsAgentEnabled ? (
          <AgentProvider
            userId={user?.id || undefined}
            channelKey={user?.id ? `insights:${user.id}` : undefined}
            transport={transport}
            debug={false}
          >
            <InsightsChatProvider>
              <ActiveThreadBridge
                activeTabId={activeTabId}
                getAgentThreadIdForTab={getAgentThreadIdForTab}
              />
              {providerChildren}
            </InsightsChatProvider>
          </AgentProvider>
        ) : (
          providerChildren
        )}
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

function ActiveThreadBridge({
  activeTabId,
  getAgentThreadIdForTab,
}: {
  activeTabId: string;
  getAgentThreadIdForTab: (tabId: string) => string;
}) {
  const { currentThreadId, setCurrentThreadId } = useInsightsChatProvider();
  const targetThreadId = useMemo(
    () => getAgentThreadIdForTab(activeTabId),
    [activeTabId, getAgentThreadIdForTab]
  );

  useEffect(() => {
    if (currentThreadId !== targetThreadId) {
      try {
        setCurrentThreadId(targetThreadId);
      } catch {}
    }
  }, [currentThreadId, targetThreadId, setCurrentThreadId]);

  return null;
}

function isQueryTab(tabId: string): boolean {
  return tabId !== HOME_TAB.id && tabId !== TEMPLATES_TAB.id;
}
