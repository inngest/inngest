// @vitest-environment jsdom
import { act, renderHook, waitFor } from '@testing-library/react';
import { useCallback, useMemo, useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { InsightsQueryStatement } from '@/gql/graphql';
import {
  openQueryTabInState,
  type TabManagerActions,
  type TabsState,
} from './InsightsTabManager/InsightsTabManager';
import { HOME_TAB } from './InsightsTabManager/constants';
import { validateInsightsSearch } from './insightsSearchParams';
import { useInsightsDeepLinkCoordinator } from './useInsightsDeepLinkCoordinator';

const mocks = vi.hoisted(() => ({
  isSavedQueriesFetching: false,
  queries: undefined as InsightsQueryStatement[] | undefined,
  queryError: undefined as string | undefined,
  toastError: vi.fn(),
}));

vi.mock('./QueryHelperPanel/StoredQueriesContext', () => ({
  useStoredQueries: () => ({
    isSavedQueriesFetching: mocks.isSavedQueriesFetching,
    queries: {
      data: mocks.queries,
      error: mocks.queryError,
      isLoading: mocks.isSavedQueriesFetching,
    },
  }),
}));

vi.mock('sonner', () => ({
  toast: { error: mocks.toastError },
}));

const SAVED_A = savedQuery('saved-A', 'Query A', 'SELECT 11');
const SAVED_B = savedQuery('saved-B', 'Query B', 'SELECT 29');
const SAVED_C = savedQuery('saved-C', 'Query C', 'SELECT 47');

function savedQuery(
  id: string,
  name: string,
  sql: string,
): InsightsQueryStatement {
  return {
    createdAt: '2026-09-01T00:00:00Z',
    creator: 'test@example.com',
    id,
    lastEditor: 'test@example.com',
    name,
    shared: false,
    sql,
    updatedAt: '2026-09-02T00:00:00Z',
  };
}

function savedTab(query: InsightsQueryStatement, tabID = `tab-${query.id}`) {
  return {
    id: tabID,
    name: query.name,
    query: query.sql,
    savedQueryId: query.id,
  };
}

function searchFromHref(href: string): Record<string, string> {
  return Object.fromEntries(
    new URL(href, 'https://app.inngest.local').searchParams,
  );
}

function useCoordinatorHarness({
  initialHref,
  initialTabState,
  initialHydrated = true,
}: {
  initialHref: string;
  initialTabState: TabsState;
  initialHydrated?: boolean;
}) {
  const [currentHref, setCurrentHref] = useState(initialHref);
  const [isHydrated, setIsHydrated] = useState(initialHydrated);
  const [tabState, setTabState] = useState(initialTabState);
  const [openCount, setOpenCount] = useState(0);
  const navigate = useCallback(
    ({ href }: { href: string; replace?: boolean }) => setCurrentHref(href),
    [],
  );
  const actions = useMemo<TabManagerActions>(
    () => ({
      breakQueryAssociation: () => {},
      closeTab: () => {},
      createNewTab: () => {},
      createTabFromQuery: () => {},
      focusTab: () => {},
      openQueryTab: (tab) => {
        setOpenCount((count) => count + 1);
        setTabState((state) => openQueryTabInState(state, tab));
      },
      openTemplatesTab: () => {},
      updateTab: () => {},
    }),
    [],
  );
  const activeTab = tabState.tabs.find(
    (tab) => tab.id === tabState.activeTabId,
  );
  const intent = validateInsightsSearch(searchFromHref(currentHref)).intent;
  const coordinator = useInsightsDeepLinkCoordinator({
    actions,
    activeTab,
    currentHref,
    intent,
    isHydrated,
    navigate,
  });

  return {
    ...coordinator,
    activeTab,
    currentHref,
    navigateTo: setCurrentHref,
    openCount,
    focusTab: (tabID: string) =>
      setTabState((state) => ({ ...state, activeTabId: tabID })),
    setIsHydrated,
    tabState,
  };
}

describe('useInsightsDeepLinkCoordinator', () => {
  beforeEach(() => {
    mocks.isSavedQueriesFetching = false;
    mocks.queries = [SAVED_A, SAVED_B, SAVED_C];
    mocks.queryError = undefined;
    mocks.toastError.mockReset();
  });

  it('replaces active A with incoming B only after B becomes active', async () => {
    const { result } = renderHook(() =>
      useCoordinatorHarness({
        initialHref:
          '/env/production/insights?query_id=saved-B&intent=%7B%22kind%22%3A%22saved-query%22%7D&keep=53',
        initialTabState: {
          tabs: [HOME_TAB, savedTab(SAVED_A)],
          activeTabId: 'tab-saved-A',
        },
      }),
    );

    await waitFor(() => expect(result.current.status).toBe('applied'));

    expect(result.current.activeTab).toMatchObject({ savedQueryId: 'saved-B' });
    expect(result.current.currentHref).toBe(
      '/env/production/insights?keep=53&query_id=saved-B',
    );
    expect(result.current.tabState.tabs).toHaveLength(3);
  });

  it('opens incoming B from an initial HOME tab without removing its permalink', async () => {
    const { result } = renderHook(() =>
      useCoordinatorHarness({
        initialHref: '/env/production/insights?query_id=saved-B',
        initialTabState: { tabs: [HOME_TAB], activeTabId: HOME_TAB.id },
      }),
    );

    await waitFor(() => expect(result.current.status).toBe('applied'));

    expect(result.current.activeTab).toMatchObject({ savedQueryId: 'saved-B' });
    expect(result.current.currentHref).toBe(
      '/env/production/insights?query_id=saved-B',
    );
  });

  it('focuses an already restored B without creating a duplicate tab', async () => {
    const restoredB = savedTab(SAVED_B, 'restored-B');
    const { result } = renderHook(() =>
      useCoordinatorHarness({
        initialHref: '/env/production/insights?query_id=saved-B',
        initialTabState: {
          tabs: [HOME_TAB, savedTab(SAVED_A), restoredB],
          activeTabId: 'tab-saved-A',
        },
      }),
    );

    await waitFor(() => expect(result.current.status).toBe('applied'));

    expect(result.current.activeTab?.id).toBe('restored-B');
    expect(result.current.tabState.tabs).toHaveLength(3);
    expect(result.current.currentHref).toBe(
      '/env/production/insights?query_id=saved-B',
    );
  });

  it('processes same-mounted navigation from B to C', async () => {
    const { result } = renderHook(() =>
      useCoordinatorHarness({
        initialHref: '/env/production/insights?query_id=saved-B',
        initialTabState: { tabs: [HOME_TAB], activeTabId: HOME_TAB.id },
      }),
    );
    await waitFor(() => expect(result.current.status).toBe('applied'));

    act(() => {
      result.current.navigateTo(
        '/env/production/insights?query_id=saved-C&keep=61',
      );
    });
    await waitFor(() =>
      expect(result.current.activeTab?.savedQueryId).toBe('saved-C'),
    );

    expect(result.current.currentHref).toBe(
      '/env/production/insights?keep=61&query_id=saved-C',
    );
    expect(result.current.openCount).toBe(2);
  });

  it('synchronizes the URL from the committed active saved tab', async () => {
    const restoredA = savedTab(SAVED_A);
    const { result } = renderHook(() =>
      useCoordinatorHarness({
        initialHref: '/env/production/insights?query_id=saved-B&keep=73',
        initialTabState: {
          tabs: [HOME_TAB, restoredA],
          activeTabId: restoredA.id,
        },
      }),
    );
    await waitFor(() => expect(result.current.status).toBe('applied'));

    act(() => result.current.focusTab(restoredA.id));
    await waitFor(() =>
      expect(result.current.currentHref).toBe(
        '/env/production/insights?keep=73&query_id=saved-A',
      ),
    );

    expect(result.current.activeTab?.savedQueryId).toBe('saved-A');
  });

  it('waits for hydration and saved-query resources before applying', async () => {
    mocks.isSavedQueriesFetching = true;
    mocks.queries = undefined;
    const { result, rerender } = renderHook(() =>
      useCoordinatorHarness({
        initialHref: '/env/production/insights?query_id=saved-B',
        initialTabState: { tabs: [HOME_TAB], activeTabId: HOME_TAB.id },
        initialHydrated: false,
      }),
    );

    expect(result.current.isPending).toBe(true);
    expect(result.current.openCount).toBe(0);

    act(() => result.current.setIsHydrated(true));
    expect(result.current.openCount).toBe(0);

    mocks.isSavedQueriesFetching = false;
    mocks.queries = [SAVED_B];
    rerender();
    await waitFor(() => expect(result.current.status).toBe('applied'));

    expect(result.current.activeTab?.savedQueryId).toBe('saved-B');
    expect(result.current.currentHref).toBe(
      '/env/production/insights?query_id=saved-B',
    );
  });

  it('terminates a failed saved-query fetch and removes the unresolvable ID', async () => {
    mocks.queries = undefined;
    mocks.queryError = 'upstream unavailable';
    const { result } = renderHook(() =>
      useCoordinatorHarness({
        initialHref:
          '/env/production/insights?query_id=saved-B&keep=recoverable#results',
        initialTabState: { tabs: [HOME_TAB], activeTabId: HOME_TAB.id },
      }),
    );

    await waitFor(() => expect(result.current.isPending).toBe(false));

    expect(result.current.activeTab).toEqual(HOME_TAB);
    expect(result.current.currentHref).toBe(
      '/env/production/insights?keep=recoverable#results',
    );
    expect(mocks.toastError).toHaveBeenCalledOnce();
    expect(mocks.toastError).toHaveBeenCalledWith(
      'Unable to load saved queries; please try again',
    );
  });

  it('prefills SQL once without running it, cleans command fields after commit, and can repeat later', async () => {
    const href =
      '/env/production/insights?sql=select+%27A+%26+B+%2B+C%27+as+marker+from+function_runs&name=Runs+%2F+caf%C3%A9&intent=%7B%22kind%22%3A%22sql-prefill%22%7D&keep=67#results';
    const { result } = renderHook(() =>
      useCoordinatorHarness({
        initialHref: href,
        initialTabState: { tabs: [HOME_TAB], activeTabId: HOME_TAB.id },
      }),
    );

    await waitFor(() => expect(result.current.status).toBe('idle'));

    expect(result.current.activeTab).toMatchObject({
      name: 'Runs / café',
      query: "select\n  'A & B + C' as marker\nfrom\n  function_runs",
    });
    expect(result.current.activeTab?.savedQueryId).toBeUndefined();
    expect(result.current.currentHref).toBe(
      '/env/production/insights?keep=67#results',
    );
    expect(result.current.openCount).toBe(1);

    act(() => result.current.navigateTo('/env/production/insights?keep=71'));
    act(() => result.current.navigateTo(href));
    await waitFor(() => expect(result.current.openCount).toBe(2));

    expect(result.current.activeTab?.query).toBe(
      "select\n  'A & B + C' as marker\nfrom\n  function_runs",
    );
    expect(result.current.currentHref).toBe(
      '/env/production/insights?keep=67#results',
    );
  });
});
