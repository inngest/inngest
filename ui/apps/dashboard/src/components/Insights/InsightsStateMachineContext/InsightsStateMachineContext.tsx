'use client';

import { createContext, useContext, useState, type ReactNode } from 'react';
import { useInfiniteQuery, type InfiniteData } from '@tanstack/react-query';

import { useStoredQueries } from '../QueryHelperPanel/StoredQueriesContext';
import { makeQuerySnapshot } from '../queries';
import type { InsightsFetchResult, InsightsStatus } from './types';
import { useFetchInsights } from './useFetchInsights';

interface InsightsStateMachineContextValue {
  activeQuery: string;
  data: InsightsFetchResult | undefined;
  error: null | Error;
  query: string;
  queryName: string;
  fetchMore: () => void;
  onChange: (value: string) => void;
  onNameChange: (name: string) => void;
  retry: () => void;
  runQuery: (query: string) => void;
  status: InsightsStatus;
}

const InsightsStateMachineContext = createContext<InsightsStateMachineContextValue | null>(null);

type InsightsStateMachineContextProviderProps = {
  children: ReactNode;
  onQueryChange: (query: string) => void;
  onQueryNameChange: (name: string) => void;
  query: string;
  queryName: string;
  renderChildren: boolean;
};

export function InsightsStateMachineContextProvider({
  children,
  onQueryChange,
  onQueryNameChange,
  query,
  queryName,
  renderChildren,
}: InsightsStateMachineContextProviderProps) {
  const [activeQuery, setActiveQuery] = useState<{ query: string; timestamp: number | null }>({
    query: '',
    timestamp: null,
  });
  const { fetchInsights } = useFetchInsights();
  const { saveQuerySnapshot } = useStoredQueries();

  const { data, error, fetchNextPage, isError, isFetching, isLoading, refetch } = useInfiniteQuery({
    enabled: hasActiveQuery(activeQuery.query),
    getNextPageParam,
    initialPageParam: null,
    queryKey: makeQueryKey(activeQuery.query, activeQuery.timestamp),
    queryFn: () => {
      return fetchInsights({ query: activeQuery.query, queryName }, (query, queryName) => {
        saveQuerySnapshot(makeQuerySnapshot(query, queryName));
      });
    },
    refetchOnWindowFocus: false,
    select: selectInsightsData,
  });

  const runQuery = (newQuery: string) => {
    setActiveQuery({ query: newQuery, timestamp: Date.now() });
  };

  return (
    <InsightsStateMachineContext.Provider
      value={{
        activeQuery: activeQuery.query,
        data,
        error,
        fetchMore: fetchNextPage,
        onChange: onQueryChange,
        onNameChange: onQueryNameChange,
        query,
        queryName,
        retry: refetch,
        runQuery,
        status: getInsightsStatus({ data, isError, isFetching, isLoading }),
      }}
    >
      {renderChildren ? children : null}
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
  return null; // No pagination support
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
 * The `timestamp` ensures that each "run" of a query is unique, allowing
 * users to re-run the same query and get fresh data from the server.
 */
function makeQueryKey(activeQuery: string, timestamp: number | null) {
  return ['insights', activeQuery, timestamp];
}

function selectInsightsData(
  infiniteData: InfiniteData<InsightsFetchResult, unknown>
): undefined | InsightsFetchResult {
  if (infiniteData.pages.length === 0) return undefined;

  const firstPage = infiniteData.pages[0];
  if (firstPage === undefined) {
    return undefined;
  }

  return {
    columns: firstPage.columns,
    rows: infiniteData.pages.flatMap((page) => page.rows),
  };
}
