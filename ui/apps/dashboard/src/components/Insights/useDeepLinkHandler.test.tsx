// @vitest-environment jsdom
import { renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { TabManagerActions } from './InsightsTabManager/InsightsTabManager';
import { useDeepLinkHandler } from './useDeepLinkHandler';

const mocks = vi.hoisted(() => ({
  isSavedQueriesFetching: false,
  queries: [] as Array<{ id: string; name: string; sql: string }>,
  toastError: vi.fn(),
}));

vi.mock('./QueryHelperPanel/StoredQueriesContext', () => ({
  useStoredQueries: () => ({
    isSavedQueriesFetching: mocks.isSavedQueriesFetching,
    queries: {
      data: mocks.queries,
      error: undefined,
      isLoading: mocks.isSavedQueriesFetching,
    },
  }),
}));

vi.mock('sonner', () => ({
  toast: { error: mocks.toastError },
}));

function createActions(): TabManagerActions {
  return {
    breakQueryAssociation: vi.fn(),
    closeTab: vi.fn(),
    createNewTab: vi.fn(),
    createTabFromQuery: vi.fn(),
    focusTab: vi.fn(),
    openTemplatesTab: vi.fn(),
    updateTab: vi.fn(),
  };
}

describe('useDeepLinkHandler', () => {
  beforeEach(() => {
    mocks.isSavedQueriesFetching = false;
    mocks.queries = [];
    mocks.toastError.mockReset();
  });

  it('formats and prefills SQL without requesting automatic execution', () => {
    const actions = createActions();
    const navigate = vi.fn();
    const sql = "select 'A & B + C' as marker from function_runs";

    const { result } = renderHook(() =>
      useDeepLinkHandler({
        actions,
        activeSavedQueryId: undefined,
        isHydrated: true,
        navigate,
        search: { sql, name: 'Progressive CEL search' },
      }),
    );

    expect(actions.createTabFromQuery).toHaveBeenCalledTimes(1);
    const createCall = vi.mocked(actions.createTabFromQuery).mock.calls[0];
    expect(createCall).toHaveLength(1);
    expect(createCall?.[0]).toMatchObject({
      name: 'Progressive CEL search',
      query: "select\n  'A & B + C' as marker\nfrom\n  function_runs",
      templateKind: 'time',
    });

    expect(navigate).toHaveBeenCalledTimes(1);
    const navigation = navigate.mock.calls[0]?.[0];
    expect(
      navigation.search({ sql, name: 'Progressive CEL search', keep: 'value' }),
    ).toEqual({ keep: 'value' });
    expect(navigation.replace).toBe(true);
    expect(result.current).toBe(false);
  });

  it('remains pending until an SQL deep link can be hydrated', () => {
    const actions = createActions();
    const navigate = vi.fn();
    const props = {
      actions,
      activeSavedQueryId: undefined,
      navigate,
      search: { sql: 'SELECT 17' },
    };
    const { result, rerender } = renderHook(
      ({ isHydrated }) => useDeepLinkHandler({ ...props, isHydrated }),
      { initialProps: { isHydrated: false } },
    );

    expect(result.current).toBe(true);
    expect(actions.createTabFromQuery).not.toHaveBeenCalled();

    rerender({ isHydrated: true });

    expect(result.current).toBe(false);
    expect(actions.createTabFromQuery).toHaveBeenCalledTimes(1);
  });

  it('processes a new SQL deep link while the Insights route stays mounted', () => {
    const actions = createActions();
    const navigate = vi.fn();
    const props = {
      actions,
      activeSavedQueryId: undefined,
      isHydrated: true,
      navigate,
      search: { sql: 'SELECT 17' } as Record<string, unknown>,
    };
    const { rerender } = renderHook(
      ({ search }) => useDeepLinkHandler({ ...props, search }),
      { initialProps: { search: props.search } },
    );

    rerender({ search: {} });
    rerender({ search: { sql: 'SELECT 29' } });

    expect(actions.createTabFromQuery).toHaveBeenCalledTimes(2);
    expect(
      vi.mocked(actions.createTabFromQuery).mock.calls[1]?.[0],
    ).toMatchObject({ query: 'SELECT\n  29' });
  });

  it('waits for saved queries before consuming a query_id deep link', () => {
    mocks.isSavedQueriesFetching = true;
    const actions = createActions();
    const navigate = vi.fn();
    const savedQuery = {
      id: 'saved-29',
      name: 'Asymmetric saved query',
      sql: 'SELECT 29',
    };
    const props = {
      actions,
      activeSavedQueryId: 'saved-29',
      isHydrated: true,
      navigate,
      search: { query_id: 'saved-29' },
    };
    const { result, rerender } = renderHook(() => useDeepLinkHandler(props));

    expect(result.current).toBe(true);
    expect(actions.createTabFromQuery).not.toHaveBeenCalled();

    mocks.isSavedQueriesFetching = false;
    mocks.queries = [savedQuery];
    rerender();

    expect(result.current).toBe(false);
    expect(actions.createTabFromQuery).toHaveBeenCalledWith(savedQuery);
    expect(navigate).not.toHaveBeenCalled();
  });
});
