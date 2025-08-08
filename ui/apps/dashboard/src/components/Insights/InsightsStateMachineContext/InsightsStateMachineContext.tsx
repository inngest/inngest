'use client';

import { createContext, useCallback, useContext, useState, type ReactNode } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';

import type { InsightsFetchResult, InsightsStatus } from './types';
import { useFetchInsights } from './useFetchInsights';
import { getInsightsStatus, selectInsightsData } from './utils';

const DEFAULT_QUERY = `SELECT
  toStartOfHour(toDateTime(event_ts / 1000)) AS hour,
  event_name,
  COUNT(*) AS event_count
FROM events
WHERE event_ts > 1754609958000
GROUP BY hour, event_name
ORDER BY hour DESC, event_name ASC`;

interface InsightsStateMachineContextValue {
  data: InsightsFetchResult | undefined;
  error: string | undefined;
  activeQuery: string;
  query: string;
  status: InsightsStatus;
  fetchMore: () => void;
  isEmpty: boolean;
  onChange: (value: string) => void;
  retry: () => void;
  runQuery: (query: string) => void;
}

const InsightsStateMachineContext = createContext<InsightsStateMachineContextValue | null>(null);

export function InsightsStateMachineContextProvider({ children }: { children: ReactNode }) {
  const [query, setQuery] = useState(DEFAULT_QUERY);
  const [activeQuery, setLastSentQuery] = useState('');
  const { fetchInsights } = useFetchInsights();

  const { data, error, fetchNextPage, isFetching, isLoading, isError, refetch } = useInfiniteQuery({
    queryKey: ['insights', activeQuery],
    queryFn: ({ pageParam }) => fetchInsights({ after: pageParam, first: 30, query: activeQuery }),
    enabled: activeQuery !== '',
    getNextPageParam: (lastPage) => {
      return lastPage.pageInfo.hasNextPage ? lastPage.pageInfo.endCursor : null;
    },
    initialPageParam: null as string | null,
    select: selectInsightsData,
  });

  const status = getInsightsStatus(isError, isLoading, isFetching, data);

  return (
    <InsightsStateMachineContext.Provider
      value={{
        data,
        error: error ? stringifyError(error) : undefined,
        activeQuery,
        query,
        status,
        fetchMore: fetchNextPage,
        isEmpty: query.trim() === '',
        onChange: setQuery,
        retry: refetch,
        runQuery: setLastSentQuery,
      }}
    >
      {children}
    </InsightsStateMachineContext.Provider>
  );
}

function stringifyError(error: unknown): string {
  if (error instanceof Error) return error.message;
  return 'Unknown error';
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
