import { useEffect, useRef, useState } from 'react';
import { Resizable } from '@inngest/components/Resizable/Resizable';

import { InsightsQueryTab } from './InsightsQueryTab';
import { InsightsTabsList } from './InsightsTabsList';
import { TableSchemaSidebar } from './TableSchemaSidebar';

const DEFAULT_QUERY = 'SELECT * FROM runs LIMIT 10;';

const TABS_STORAGE_KEY = 'duckdb-insights-tabs-state';

export type InsightsTabData = {
  id: string;
  name: string;
  sql: string;
};

interface TabsStorageState {
  tabs: InsightsTabData[];
  activeTabId: string;
}

function isTabsStorageState(value: unknown): value is TabsStorageState {
  if (!value || typeof value !== 'object') return false;
  const { tabs, activeTabId } = value as Partial<TabsStorageState>;
  return (
    Array.isArray(tabs) &&
    tabs.length > 0 &&
    tabs.every(
      (tab) =>
        typeof tab?.id === 'string' &&
        typeof tab?.name === 'string' &&
        typeof tab?.sql === 'string',
    ) &&
    typeof activeTabId === 'string'
  );
}

// Skips during SSR/tests where localStorage may not exist -- mirrors the
// dashboard Insights feature's InsightsTabManager.tsx getStoredTabs.
function getStoredTabs(): TabsStorageState | null {
  if (typeof window === 'undefined') return null;
  try {
    const stored = window.localStorage.getItem(TABS_STORAGE_KEY);
    if (!stored) return null;
    const parsed: unknown = JSON.parse(stored);
    return isTabsStorageState(parsed) ? parsed : null;
  } catch {
    return null;
  }
}

function saveTabsToStorage(tabs: InsightsTabData[], activeTabId: string) {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(
      TABS_STORAGE_KEY,
      JSON.stringify({ tabs, activeTabId }),
    );
  } catch {}
}

function makeTab(name: string): InsightsTabData {
  return { id: crypto.randomUUID(), name, sql: DEFAULT_QUERY };
}

// Tabs are always named "Query N"; picks the next unused N so restored tabs
// don't collide with newly created ones.
function getNextTabNumber(tabs: InsightsTabData[]): number {
  const numbers = tabs.map((tab) => {
    const match = /^Query (\d+)$/.exec(tab.name);
    return match ? Number(match[1]) : 0;
  });
  return Math.max(0, ...numbers) + 1;
}

// Picks the tab to focus after closing tabIdToClose, mirroring the dashboard
// Insights feature's InsightsTabManager.tsx's getNewActiveTabAfterClose.
function getNewActiveTabAfterClose(
  tabs: InsightsTabData[],
  tabIdToClose: string,
  activeTabId: string,
): string {
  if (tabIdToClose !== activeTabId) return activeTabId;

  const closingIndex = tabs.findIndex((tab) => tab.id === tabIdToClose);
  const remaining = tabs.filter((tab) => tab.id !== tabIdToClose);
  return (
    remaining[closingIndex]?.id ??
    remaining[closingIndex - 1]?.id ??
    remaining[0].id
  );
}

function getInitialState(): TabsStorageState {
  const stored = getStoredTabs();
  if (stored && stored.tabs.some((tab) => tab.id === stored.activeTabId)) {
    return stored;
  }
  const tab = makeTab('Query 1');
  return { tabs: [tab], activeTabId: tab.id };
}

export default function InsightsPage() {
  const [initialState] = useState(getInitialState);
  const [tabs, setTabs] = useState<InsightsTabData[]>(initialState.tabs);
  const [activeTabId, setActiveTabId] = useState(initialState.activeTabId);
  const [nextTabNumber, setNextTabNumber] = useState(() =>
    getNextTabNumber(initialState.tabs),
  );
  const isFirstRenderRef = useRef(true);

  useEffect(() => {
    if (isFirstRenderRef.current) {
      isFirstRenderRef.current = false;
      return;
    }
    saveTabsToStorage(tabs, activeTabId);
  }, [tabs, activeTabId]);

  const handleCreateTab = () => {
    const tab = makeTab(`Query ${nextTabNumber}`);
    setNextTabNumber((n) => n + 1);
    setTabs((prev) => [...prev, tab]);
    setActiveTabId(tab.id);
  };

  const handleCloseTab = (id: string) => {
    if (tabs.length <= 1) return;
    setActiveTabId((current) => getNewActiveTabAfterClose(tabs, id, current));
    setTabs((prev) => prev.filter((tab) => tab.id !== id));
  };

  const handleSqlChange = (id: string, sql: string) => {
    setTabs((prev) =>
      prev.map((tab) => (tab.id === id ? { ...tab, sql } : tab)),
    );
  };

  const tabsAndContent = (
    <div className="flex h-full min-h-0 flex-col">
      <InsightsTabsList
        activeTabId={activeTabId}
        tabs={tabs}
        onValueChange={setActiveTabId}
        onClose={handleCloseTab}
        onCreateTab={handleCreateTab}
      />
      <div className="min-h-0 flex-1">
        {tabs.map((tab) => (
          <div
            key={tab.id}
            className={
              tab.id === activeTabId
                ? 'h-full w-full'
                : 'h-0 w-full overflow-hidden'
            }
          >
            <InsightsQueryTab
              initialSql={tab.sql}
              onSqlChange={(sql) => handleSqlChange(tab.id, sql)}
            />
          </div>
        ))}
      </div>
    </div>
  );

  return (
    <Resizable
      orientation="horizontal"
      defaultSplitPercentage={85}
      minSplitPercentage={60}
      maxSplitPercentage={85}
      splitKey="insights-schema-sidebar-split"
      first={tabsAndContent}
      second={<TableSchemaSidebar />}
    />
  );
}
