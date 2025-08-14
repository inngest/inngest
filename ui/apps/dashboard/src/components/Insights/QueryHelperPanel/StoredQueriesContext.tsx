'use client';

import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';

import type { Query } from './types';

interface QueryResult<T> {
  data: T | undefined;
  error: string | undefined;
  isLoading: boolean;
}

interface StoredQueriesContextValue {
  recentQueries: QueryResult<Query[]>;
  savedQueries: QueryResult<Query[]>;
}

const StoredQueriesContext = createContext<undefined | StoredQueriesContextValue>(undefined);

export function useStoredQueries(): StoredQueriesContextValue {
  const context = useContext(StoredQueriesContext);
  if (context === undefined) {
    throw new Error('useStoredQueries must be used within a StoredQueriesProvider');
  }

  return context;
}

interface StoredQueriesProviderProps {
  children: ReactNode;
}

// TODO: Replace mock data with actual API calls
const MOCK_RECENT_QUERIES: Query[] = [
  {
    id: 'recent-query-1',
    isSavedQuery: false,
    name: 'SELECT COUNT(*) FROM events WHERE ts > NOW() - INTERVAL 1 HOUR',
    query: 'Recent query 1 query text',
  },
  {
    id: 'recent-query-2',
    isSavedQuery: false,
    name: 'SELECT status, COUNT(*) FROM function_runs GROUP BY status',
    query: 'Recent query 2 query text',
  },
];

// TODO: Replace mock data with actual API calls
const MOCK_SAVED_QUERIES: Query[] = [
  {
    id: 'saved-query-1',
    isSavedQuery: true,
    name: 'Saved Query 1',
    query: 'Saved query 1 query text',
  },
  {
    id: 'saved-query-2',
    isSavedQuery: true,
    name: 'Saved Query 2',
    query: 'Saved query 2 query text',
  },
];

function simulateAsyncLoad<T>(
  data: T,
  delay: number,
  setState: (state: QueryResult<T>) => void
): void {
  setTimeout(() => {
    if (Math.random() < 0.2) {
      setState({ data: undefined, error: 'Failed to fetch queries', isLoading: false });
    } else {
      setState({ data, error: undefined, isLoading: false });
    }
  }, delay);
}

export function StoredQueriesProvider({ children }: StoredQueriesProviderProps) {
  const [recentQueries, setRecentQueries] = useState<QueryResult<Query[]>>({
    data: undefined,
    error: undefined,
    isLoading: true,
  });

  const [savedQueries, setSavedQueries] = useState<QueryResult<Query[]>>({
    data: undefined,
    error: undefined,
    isLoading: true,
  });

  useEffect(() => {
    simulateAsyncLoad(MOCK_RECENT_QUERIES, 1000, setRecentQueries);
  }, []);

  useEffect(() => {
    simulateAsyncLoad(MOCK_SAVED_QUERIES, 500, setSavedQueries);
  }, []);

  const value: StoredQueriesContextValue = { recentQueries, savedQueries };

  return <StoredQueriesContext.Provider value={value}>{children}</StoredQueriesContext.Provider>;
}
