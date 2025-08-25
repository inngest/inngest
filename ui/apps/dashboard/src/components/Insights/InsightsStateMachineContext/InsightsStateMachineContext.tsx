'use client';

import { createContext, useCallback, useContext, useState, type ReactNode } from 'react';
import { useQuery } from '@tanstack/react-query';

import { useStoredQueries } from '../QueryHelperPanel/StoredQueriesContext';
import { makeQuerySnapshot } from '../queries';
import type { InsightsFetchResult, InsightsStatus } from './types';
import { useFetchInsights } from './useFetchInsights';

interface InsightsStateMachineContextValue {
  data: InsightsFetchResult | undefined;
  error: null | Error;
  query: string;
  queryName: string;
  onChange: (value: string) => void;
  onNameChange: (name: string) => void;
  retry: () => void;
  runQuery: () => void;
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
  tabId: string;
};

export function InsightsStateMachineContextProvider({
  children,
  onQueryChange,
  onQueryNameChange,
  query,
  queryName,
  renderChildren,
  tabId,
}: InsightsStateMachineContextProviderProps) {
  const { fetchInsights } = useFetchInsights();
  const { saveQuerySnapshot } = useStoredQueries();

  const { data, error, isError, isFetching, refetch } = useQuery({
    enabled: false,
    gcTime: 0,
    queryKey: ['insights', tabId],
    queryFn: () => {
      return fetchInsights({ query, queryName }, (query, queryName) => {
        saveQuerySnapshot(makeQuerySnapshot(query, queryName));
      });
    },
    staleTime: 0,
  });

  const runQuery = useCallback(() => {
    refetch();
  }, [refetch]);

  return (
    <InsightsStateMachineContext.Provider
      value={{
        data,
        error,
        onChange: onQueryChange,
        onNameChange: onQueryNameChange,
        query,
        queryName,
        retry: refetch,
        runQuery,
        status: getInsightsStatus({ data, isError, isLoading: isFetching }),
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
