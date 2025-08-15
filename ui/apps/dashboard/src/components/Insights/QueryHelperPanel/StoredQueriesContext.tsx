'use client';

import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';
import { useLocalStorage } from 'react-use';

import type { Query, QuerySnapshot, UnsavedQuery } from '@/components/Insights/types';
import { MOCK_QUERY_SNAPSHOTS, MOCK_SAVED_QUERIES } from './mocks';

type ID = string;
type QueryRecord<T> = Record<ID, T>;

const EMPTY_QUERY_SNAPSHOTS: QueryRecord<QuerySnapshot> = {};

interface StoredQueriesContextValue {
  addUnsavedQuery: (query: UnsavedQuery) => void;
  queries: QueryRecord<Query>;
  querySnapshots: QueryRecord<QuerySnapshot>;
  removeUnsavedQuery: (id: ID) => void;
  saveQuery: (query: Query) => void;
  saveQuerySnapshot: (snapshot: QuerySnapshot) => void;
}

const StoredQueriesContext = createContext<undefined | StoredQueriesContextValue>(undefined);

interface StoredQueriesProviderProps {
  children: ReactNode;
}

export function StoredQueriesProvider({ children }: StoredQueriesProviderProps) {
  const [querySnapshots, setQuerySnapshots] = useLocalStorage<QueryRecord<QuerySnapshot>>(
    'insights-query-snapshots',
    MOCK_QUERY_SNAPSHOTS
  );

  const [savedQueries, setSavedQueries] = useLocalStorage<QueryRecord<Query>>(
    'insights-saved-queries',
    MOCK_SAVED_QUERIES
  );

  const [unsavedQueries, setUnsavedQueries] = useState<QueryRecord<Query>>({});

  const addUnsavedQuery = useCallback((query: UnsavedQuery) => {
    setUnsavedQueries((prev) => ({
      ...prev,
      [query.id]: { ...query, saved: false },
    }));
  }, []);

  const removeUnsavedQuery = useCallback((id: ID) => {
    setUnsavedQueries((prev) => {
      const newQueries = { ...prev };
      delete { ...prev }[id];
      return newQueries;
    });
  }, []);

  const saveQuery = useCallback(
    (query: Query) => {
      const savedQuery = { ...query, saved: true };
      setSavedQueries((prev) => ({ ...prev, [query.id]: savedQuery }));

      setUnsavedQueries((prev) => {
        const newQueries = { ...prev };
        delete { ...prev }[query.id];
        return newQueries;
      });
    },
    [setSavedQueries]
  );

  const saveQuerySnapshot = useCallback(
    (snapshot: QuerySnapshot) => {
      setQuerySnapshots((prev) => ({ ...prev, [snapshot.id]: snapshot }));
    },
    [setQuerySnapshots]
  );

  const queries = useMemo(() => {
    return { ...unsavedQueries, ...savedQueries };
  }, [unsavedQueries, savedQueries]);

  const contextValue = useMemo(
    () => ({
      queries,
      querySnapshots: querySnapshots ?? EMPTY_QUERY_SNAPSHOTS,
      addUnsavedQuery,
      removeUnsavedQuery,
      saveQuery,
      saveQuerySnapshot,
    }),
    [querySnapshots, queries, addUnsavedQuery, removeUnsavedQuery, saveQuery, saveQuerySnapshot]
  );

  return (
    <StoredQueriesContext.Provider value={contextValue}>{children}</StoredQueriesContext.Provider>
  );
}

export function useStoredQueries(): StoredQueriesContextValue {
  const context = useContext(StoredQueriesContext);
  if (context === undefined) {
    throw new Error('useStoredQueries must be used within a StoredQueriesProvider');
  }

  return context;
}
