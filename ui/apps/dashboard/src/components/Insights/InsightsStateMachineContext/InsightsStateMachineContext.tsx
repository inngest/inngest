'use client';

import { createContext, useContext, useState, type ReactNode } from 'react';
import { useQuery } from '@tanstack/react-query';

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

  const { data, error, isError, isLoading, refetch } = useQuery({
    enabled: hasActiveQuery(activeQuery.query),
    queryKey: makeQueryKey(activeQuery.query, activeQuery.timestamp),
    queryFn: () => {
      return fetchInsights({ query: activeQuery.query, queryName }, (query, queryName) => {
        saveQuerySnapshot(makeQuerySnapshot(query, queryName));
      });
    },
    refetchOnWindowFocus: false,
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
        onChange: onQueryChange,
        onNameChange: onQueryNameChange,
        query,
        queryName,
        retry: refetch,
        runQuery,
        status: getInsightsStatus({ data, isError, isLoading }),
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
  isLoading: boolean;
}

export function getInsightsStatus({
  data,
  isError,
  isLoading,
}: GetInsightsStatusParams): InsightsStatus {
  if (isError) return 'error';
  if (isLoading) return 'loading';
  if (data !== undefined) return 'success';
  return 'initial';
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
