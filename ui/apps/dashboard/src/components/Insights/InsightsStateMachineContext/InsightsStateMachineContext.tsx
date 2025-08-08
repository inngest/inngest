'use client';

import { createContext, useContext, useState, type ReactNode } from 'react';
import { useInfiniteQuery, type InfiniteData } from '@tanstack/react-query';

import type { InsightsFetchResult, InsightsStatus } from './types';
import { useFetchInsights } from './useFetchInsights';

const DEFAULT_QUERY = `SELECT
  toStartOfHour(toDateTime(event_ts / 1000)) AS hour,
  event_name,
  COUNT(*) AS event_count
FROM events
WHERE event_ts > 1754609958000
GROUP BY hour, event_name
ORDER BY hour DESC, event_name ASC`;

interface InsightsStateMachineContextValue {
  activeQuery: string;
  data: InsightsFetchResult | undefined;
  error: null | Error;
  query: string;
  fetchMore: () => void;
  onChange: (value: string) => void;
  retry: () => void;
  runQuery: (query: string) => void;
  status: InsightsStatus;
}

const InsightsStateMachineContext = createContext<InsightsStateMachineContextValue | null>(null);

export function InsightsStateMachineContextProvider({ children }: { children: ReactNode }) {
  const [query, setQuery] = useState(DEFAULT_QUERY);
  const [activeQuery, setActiveQuery] = useState('');
  const { fetchInsights } = useFetchInsights();

  const { data, error, fetchNextPage, isError, isFetching, isLoading, refetch } = useInfiniteQuery({
    enabled: hasActiveQuery(activeQuery),
    getNextPageParam,
    initialPageParam: null,
    queryKey: makeQueryKey(activeQuery),
    queryFn: ({ pageParam }) => fetchInsights({ after: pageParam, first: 30, query: activeQuery }),
    select: selectInsightsData,
  });

  return (
    <InsightsStateMachineContext.Provider
      value={{
        activeQuery,
        data,
        error,
        fetchMore: fetchNextPage,
        onChange: setQuery,
        query,
        retry: refetch,
        runQuery: setActiveQuery,
        status: getInsightsStatus({ data, isError, isFetching, isLoading }),
      }}
    >
      {children}
    </InsightsStateMachineContext.Provider>
  );
}

export function useInsightsStateMachineContext() {
  const context = useContext(InsightsStateMachineContext);
  if (!context) {
    throw new Error(
      'useInsightsStateMachineContext must be used within InsightsStateMachineContextProvider'
    );
  }

  return context;
}

interface GetInsightsStatusParams {
  data: undefined | InsightsFetchResult;
  isError: boolean;
  isFetching: boolean;
  isLoading: boolean;
}

export function getInsightsStatus({
  data,
  isError,
  isFetching,
  isLoading,
}: GetInsightsStatusParams): InsightsStatus {
  if (isError && data === undefined) return 'error';
  if (isError && data !== undefined) return 'fetchMoreError';
  if (isLoading) return 'loading';
  if (isFetching && data !== undefined) return 'fetchingMore';
  if (data !== undefined) return 'success';
  return 'initial';
}

function getNextPageParam(lastPage: InsightsFetchResult) {
  return lastPage.pageInfo.hasNextPage ? lastPage.pageInfo.endCursor : null;
}

/**
 * This prevents the query from fetching until the user runs a query,
 * since `activeQuery` is not updated until that button is clicked.
 */
function hasActiveQuery(activeQuery: string) {
  return activeQuery.trim() !== '';
}

/**
 * `activeQuery` changes only when the user runs a new query.
 * Thus, a new fetch will be triggered when the user clicks the "Run query"
 * button, but not automatically when the user types something else in the editor.
 */
function makeQueryKey(activeQuery: string) {
  return ['insights', activeQuery];
}

function selectInsightsData(
  infiniteData: InfiniteData<InsightsFetchResult, unknown>
): undefined | InsightsFetchResult {
  if (!infiniteData?.pages?.length) return undefined;

  const firstPage = infiniteData.pages[0];
  const lastPage = infiniteData.pages[infiniteData.pages.length - 1];
  if (firstPage === undefined || lastPage === undefined) {
    return undefined;
  }

  return {
    columns: firstPage.columns,
    entries: infiniteData.pages.flatMap((page) => page.entries),
    pageInfo: lastPage.pageInfo,
    totalCount: firstPage.totalCount,
  };
}
