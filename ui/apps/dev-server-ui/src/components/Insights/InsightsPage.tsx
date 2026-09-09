import { useEffect, useRef, useState } from 'react';
import {
  HelperPanelControl,
  HelperPanelFrame,
  type HelperItem,
} from '@inngest/components/HelperPanelControl';
import { Resizable } from '@inngest/components/Resizable/Resizable';
import { RiFunctionLine, RiNodeTree, RiTableLine } from '@remixicon/react';

import { CellDetail, type CellDetailData } from './CellDetail';
import { InsightsQueryTab } from './InsightsQueryTab';
import { InsightsTabsList } from './InsightsTabsList';
import { FunctionsSidebar, TableSchemaSidebar } from './TableSchemaSidebar';

const DEFAULT_QUERY = 'SELECT * FROM runs LIMIT 10;';

const TABS_STORAGE_KEY = 'duckdb-insights-tabs-state';

// Mirrors the dashboard Insights feature's InsightsHelperPanel/constants.ts:
// one shared side panel, switched between a fixed set of named views rather
// than several simultaneously-open panes. Schema and Functions each have
// their own entry in the vertical icon bar (HelperPanelControl); Cell Detail
// doesn't -- it's only reached by clicking a result cell (see
// handleSelectedCellChange below), exactly like the dashboard's own
// CELL_DETAIL helper.
const SCHEMA = 'Schema' as const;
const FUNCTIONS = 'Functions' as const;
const CELL_DETAIL = 'Cell Detail' as const;
type HelperTitle = typeof SCHEMA | typeof FUNCTIONS | typeof CELL_DETAIL;

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
  // Keyed by tab id rather than a single value, so switching tabs doesn't
  // show one tab's selected cell over another's results -- each
  // InsightsQueryTab instance stays mounted (just hidden) while inactive,
  // so its own entry here persists across tab switches too.
  const [selectedCellByTab, setSelectedCellByTab] = useState<
    Record<string, CellDetailData | null>
  >({});
  const [activeHelper, setActiveHelper] = useState<HelperTitle | null>(null);
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
    setSelectedCellByTab((prev) => {
      if (!(id in prev)) return prev;
      const { [id]: _removed, ...rest } = prev;
      return rest;
    });
  };

  const handleSqlChange = (id: string, sql: string) => {
    setTabs((prev) =>
      prev.map((tab) => (tab.id === id ? { ...tab, sql } : tab)),
    );
  };

  // Selecting a cell always switches the shared panel to Cell Detail,
  // replacing whatever was open (e.g. Schema) -- mirrors the dashboard's
  // CellDetailProvider onOpenPanel -> setActiveHelper(CELL_DETAIL).
  // Deselecting (Escape, or clicking the same cell's row out of view)
  // leaves the panel open on Cell Detail showing its "click a cell" state,
  // same as the dashboard.
  const handleSelectedCellChange = (
    tabId: string,
    cell: CellDetailData | null,
  ) => {
    setSelectedCellByTab((prev) => ({ ...prev, [tabId]: cell }));
    if (cell) setActiveHelper(CELL_DETAIL);
  };

  const handleToggleHelper = (title: HelperTitle) => {
    setActiveHelper((current) => (current === title ? null : title));
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
              selectedCell={selectedCellByTab[tab.id] ?? null}
              onSelectedCellChange={(cell) =>
                handleSelectedCellChange(tab.id, cell)
              }
            />
          </div>
        ))}
      </div>
    </div>
  );

  const activeSelectedCell = selectedCellByTab[activeTabId] ?? null;
  const isHelperPanelOpen = activeHelper !== null;

  const helperItems: HelperItem[] = [
    {
      title: SCHEMA,
      icon: <RiNodeTree size={20} />,
      action: () => handleToggleHelper(SCHEMA),
    },
    {
      title: FUNCTIONS,
      icon: <RiFunctionLine size={20} />,
      action: () => handleToggleHelper(FUNCTIONS),
    },
  ];

  // One shared panel switched between named views, rather than several
  // simultaneously-open panes -- mirrors the dashboard Insights feature's
  // InsightsTabManager.tsx (Resizable(mainContent, HelperPanelFrame) plus
  // an always-visible HelperPanelControl icon bar outside it).
  const mainContent = isHelperPanelOpen ? (
    <Resizable
      orientation="horizontal"
      defaultSplitPercentage={78}
      minSplitPercentage={50}
      maxSplitPercentage={85}
      splitKey="insights-helper-panel-split"
      first={tabsAndContent}
      second={
        <HelperPanelFrame
          title={activeHelper}
          icon={
            activeHelper === SCHEMA ? (
              <RiNodeTree size={20} className="text-subtle" />
            ) : activeHelper === FUNCTIONS ? (
              <RiFunctionLine size={20} className="text-subtle" />
            ) : (
              <RiTableLine size={20} className="text-subtle" />
            )
          }
          onClose={() => setActiveHelper(null)}
          contentClassName="overflow-hidden"
        >
          {activeHelper === SCHEMA ? (
            <TableSchemaSidebar />
          ) : activeHelper === FUNCTIONS ? (
            <FunctionsSidebar />
          ) : (
            <CellDetail selectedCell={activeSelectedCell} />
          )}
        </HelperPanelFrame>
      }
    />
  ) : (
    tabsAndContent
  );

  return (
    <div className="flex h-full w-full">
      <div className="h-full min-w-0 flex-1 overflow-hidden">{mainContent}</div>
      <HelperPanelControl items={helperItems} activeTitle={activeHelper} />
    </div>
  );
}
