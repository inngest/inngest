import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";

import { Resizable } from "@inngest/components/Resizable/Resizable";
import {
  AgentProvider,
  createInMemorySessionTransport,
} from "@inngest/use-agent";
import { ulid } from "ulid";
import { v4 as uuidv4 } from "uuid";

import { useBooleanFlag } from "@/components/FeatureFlags/hooks";
import { InsightsStateMachineContextProvider } from "@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext";
import type {
  QuerySnapshot,
  QueryTemplate,
  Tab,
} from "@/components/Insights/types";
import type { InsightsQueryStatement } from "@/gql/graphql";
import { pathCreator } from "@/utils/urls";
import { isQuerySnapshot, isQueryTemplate } from "../queries";
import { SHOW_DOCS_CONTROL_PANEL_BUTTON } from "../temp-flags";
import { InsightsHelperPanel } from "./InsightsHelperPanel/InsightsHelperPanel";
import {
  InsightsHelperPanelControl,
  type HelperItem,
} from "./InsightsHelperPanel/InsightsHelperPanelControl";
import { InsightsHelperPanelIcon } from "./InsightsHelperPanel/InsightsHelperPanelIcon";
import {
  DOCUMENTATION,
  INSIGHTS_AI,
  SCHEMA_EXPLORER,
  SUPPORT,
  type HelperTitle,
} from "./InsightsHelperPanel/constants";
import {
  InsightsChatProvider,
  useInsightsChatProvider,
} from "./InsightsHelperPanel/features/InsightsChat/InsightsChatProvider";
import { InsightsTabPanel } from "./InsightsTabPanel";
import { InsightsTabsList } from "./InsightsTabsList";
import { HOME_TAB, TEMPLATES_TAB, UNTITLED_QUERY } from "./constants";
import { useUser } from "@clerk/tanstack-react-start";

const TABS_STORAGE_KEY = "insights-tabs-state";

interface TabsStorageState {
  tabs: Tab[];
  activeTabId: string;
}

function getStoredTabs(): TabsStorageState | null {
  // Skip during SSR - localStorage only exists in browser
  if (typeof window === "undefined") return null;
  try {
    const stored = localStorage.getItem(TABS_STORAGE_KEY);
    if (!stored) return null;
    return JSON.parse(stored);
  } catch {
    return null;
  }
}

function saveTabsToStorage(tabs: Tab[], activeTabId: string) {
  // Skip during SSR - localStorage only exists in browser
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(
      TABS_STORAGE_KEY,
      JSON.stringify({ tabs, activeTabId }),
    );
  } catch {}
}

export interface TabManagerActions {
  breakQueryAssociation: (savedQueryId: string) => void;
  closeTab: (id: string) => void;
  createNewTab: () => void;
  createTabFromQuery: (
    query: InsightsQueryStatement | QuerySnapshot | QueryTemplate,
  ) => void;
  focusTab: (id: string) => void;
  openTemplatesTab: () => void;
  updateTab: (id: string, patch: Partial<Omit<Tab, "id">>) => void;
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
  props: UseInsightsTabManagerProps,
): UseInsightsTabManagerReturn {
  const [isMounted, setIsMounted] = useState(false);
  const [tabs, setTabs] = useState<Tab[]>([HOME_TAB]);
  const [activeTabId, setActiveTabId] = useState<string>(HOME_TAB.id);
  const isInsightsAgentEnabled = useBooleanFlag("insights-agent");
  const isSchemaWidgetEnabled = useBooleanFlag("insights-schema-widget");

  // Load from localStorage after mount (avoids hydration mismatch)
  useEffect(() => {
    const stored = getStoredTabs();
    if (stored) {
      setTabs(stored.tabs);
      setActiveTabId(stored.activeTabId);
    }
    setIsMounted(true);
  }, []);

  // Save tabs to local storage whenever they change (only after initial mount)
  useEffect(() => {
    if (isMounted) {
      saveTabsToStorage(tabs, activeTabId);
    }
  }, [tabs, activeTabId, isMounted]);

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
    [setActiveTabId],
  );

  const actions = useMemo(
    () => ({
      breakQueryAssociation: (savedQueryId: string) => {
        setTabs((prevTabs) => {
          // Find the tab associated with this savedQueryId
          const tabToClose = prevTabs.find(
            (tab) => tab.savedQueryId === savedQueryId,
          );

          if (tabToClose) {
            // Close the tab entirely when query is deleted
            const newTabs = prevTabs.filter((tab) => tab.id !== tabToClose.id);
            const newActiveTabId = getNewActiveTabAfterClose(
              prevTabs,
              tabToClose.id,
              activeTabId,
            );
            if (newActiveTabId !== undefined) setActiveTabId(newActiveTabId);
            return newTabs;
          }

          return prevTabs;
        });
      },
      closeTab: (id: string) => {
        setTabs((prevTabs) => {
          const newTabs = prevTabs.filter((tab) => tab.id !== id);

          const newActiveTabId = getNewActiveTabAfterClose(
            prevTabs,
            id,
            activeTabId,
          );
          if (newActiveTabId !== undefined) setActiveTabId(newActiveTabId);

          return newTabs;
        });
      },
      createNewTab: () => {
        createTabBase(makeEmptyUnsavedTab());
      },
      createTabFromQuery: (
        query: InsightsQueryStatement | QuerySnapshot | QueryTemplate,
      ) => {
        if (isQueryTemplate(query)) {
          createTabBase({
            ...makeEmptyUnsavedTab(),
            query: query.query,
            name: query.name,
          });
          return;
        }

        if (isQuerySnapshot(query)) {
          createTabBase({ ...makeEmptyUnsavedTab(), query: query.query });
          return;
        }

        const tabWithSameSavedQueryId = tabs.find(
          (tab) => tab.savedQueryId === query.id,
        );
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
      updateTab: (id: string, tab: Partial<Omit<Tab, "id">>) => {
        setTabs((prevTabs) =>
          prevTabs.map((t) => (t.id === id ? { ...t, ...tab } : t)),
        );
      },
    }),
    [activeTabId, createTabBase, tabs],
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
        onToggleQueryHelperPanelVisibility={
          props.onToggleQueryHelperPanelVisibility
        }
        isInsightsAgentEnabled={isInsightsAgentEnabled.value}
        isSchemaWidgetEnabled={isSchemaWidgetEnabled.value}
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
      isInsightsAgentEnabled.value,
      isSchemaWidgetEnabled.value,
    ],
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
  isInsightsAgentEnabled: boolean;
  isSchemaWidgetEnabled: boolean;
}

// TODO: Remove check on isInsightsAgentEnabled to determine whether to render InsightsHelperPanelControl.
// That check currently exists because most customers would only see the support link icon, which would be strange.
function InsightsTabManagerInternal({
  tabs,
  activeTabId,
  actions,
  getAgentThreadIdForTab,
  historyWindow,
  isQueryHelperPanelVisible,
  onToggleQueryHelperPanelVisibility,
  isInsightsAgentEnabled,
  isSchemaWidgetEnabled,
}: InsightsTabManagerInternalProps) {
  const [activeHelper, setActiveHelper] = useState<HelperTitle | null>(null);

  const handleSelectHelper = useCallback(
    (title: HelperTitle) => {
      if (activeHelper === title) {
        setActiveHelper(null);
      } else {
        setActiveHelper(title);
      }
    },
    [activeHelper],
  );

  const isHelperPanelOpen = activeHelper !== null;

  const helperItems = useMemo<HelperItem[]>(() => {
    const items: HelperItem[] = [];

    if (isInsightsAgentEnabled) {
      items.push({
        title: INSIGHTS_AI,
        icon: <InsightsHelperPanelIcon title={INSIGHTS_AI} />,
        action: () => handleSelectHelper(INSIGHTS_AI),
      });
    }

    if (SHOW_DOCS_CONTROL_PANEL_BUTTON) {
      items.push({
        title: DOCUMENTATION,
        icon: <InsightsHelperPanelIcon title={DOCUMENTATION} />,
        action: () => handleSelectHelper(DOCUMENTATION),
      });
    }

    if (isSchemaWidgetEnabled) {
      items.push({
        title: SCHEMA_EXPLORER,
        icon: <InsightsHelperPanelIcon title={SCHEMA_EXPLORER} />,
        action: () => handleSelectHelper(SCHEMA_EXPLORER),
      });
    }

    items.push({
      title: SUPPORT,
      icon: <InsightsHelperPanelIcon title={SUPPORT} />,
      action: noOp,
      href: pathCreator.support({ ref: "app-insights" }),
    });

    return items;
  }, [handleSelectHelper, isInsightsAgentEnabled, isSchemaWidgetEnabled]);
  // Provide shared transport/connection for all descendant useAgents hooks
  const { user } = useUser();
  const transport = useMemo(
    () =>
      isInsightsAgentEnabled ? createInMemorySessionTransport() : undefined,
    [isInsightsAgentEnabled],
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
          <div
            className={
              tab.id === activeTabId
                ? "h-full w-full"
                : "h-0 w-full overflow-hidden"
            }
          >
            <div className="flex h-full w-full">
              <div className="h-full min-w-0 flex-1 overflow-hidden">
                {isQueryTab(tab.id) && isHelperPanelOpen ? (
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
                    second={
                      <InsightsHelperPanel
                        active={activeHelper}
                        agentThreadId={getAgentThreadIdForTab(tab.id)}
                        onClose={() => {
                          setActiveHelper(null);
                        }}
                      />
                    }
                  />
                ) : (
                  <InsightsTabPanel
                    isHomeTab={tab.id === HOME_TAB.id}
                    isTemplatesTab={tab.id === TEMPLATES_TAB.id}
                    tab={tab}
                    historyWindow={historyWindow}
                  />
                )}
              </div>
              {isQueryTab(tab.id) &&
              hasMoreThanOneHelperPanelFeatureEnabled(helperItems) ? (
                <InsightsHelperPanelControl
                  items={helperItems}
                  activeTitle={activeHelper}
                />
              ) : null}
            </div>
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

// This ensures the user has support + at least one of AI, Documentation, or Schema Explorer enabled.
// Otherwise, we just hide the helper panel because only showing support is not useful.
function hasMoreThanOneHelperPanelFeatureEnabled(
  features: HelperItem[],
): boolean {
  return features.length > 1;
}

function getNewActiveTabAfterClose(
  existingTabs: Tab[],
  tabIdToClose: string,
  currentActiveTabId: string,
): undefined | string {
  if (tabIdToClose !== currentActiveTabId) return currentActiveTabId;

  const closingTabIndex = existingTabs.findIndex(
    (tab) => tab.id === tabIdToClose,
  );
  if (closingTabIndex === -1) return currentActiveTabId;

  // 1: Try to select the next tab (now where the closed tab was).
  // 2: Try to select the tab before the closed tab.
  const remainingTabs = existingTabs.filter((tab) => tab.id !== tabIdToClose);
  const newlySelectedTabId =
    remainingTabs[closingTabIndex]?.id ??
    remainingTabs[closingTabIndex - 1]?.id;
  return newlySelectedTabId;
}

export function hasDiffWithSavedQuery(
  savedQueries: InsightsQueryStatement[] | undefined,
  tab: Tab,
): boolean {
  if (tab.savedQueryId === undefined || savedQueries === undefined)
    return false;
  const savedQuery = savedQueries.find((q) => q.id === tab.savedQueryId);
  if (!savedQuery) return false;
  return savedQuery.name !== tab.name || savedQuery.sql !== tab.query;
}

/**
 * Determines if a tab has unsaved changes by comparing against either:
 * - The saved query state (if tab.savedQueryId exists), or
 * - The blank state (for new unsaved queries)
 */
export function hasUnsavedChanges(
  savedQueries: InsightsQueryStatement[] | undefined,
  tab: Tab,
): boolean {
  // If tab is associated with a saved query, check diff with saved state
  if (tab.savedQueryId !== undefined) {
    return hasDiffWithSavedQuery(savedQueries, tab);
  }

  // For new queries, check if there's any content different from blank state
  return tab.name !== UNTITLED_QUERY || tab.query !== "";
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
  return { id: ulid(), name: UNTITLED_QUERY, query: "" };
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
    [activeTabId, getAgentThreadIdForTab],
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

function noOp() {
  return;
}
