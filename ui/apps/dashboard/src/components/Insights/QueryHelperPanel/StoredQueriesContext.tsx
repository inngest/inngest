'use client';

import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';
import { useLocalStorage } from 'react-use';

import type { TabManagerActions } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import type { Query, QuerySnapshot, UnsavedQuery } from '@/components/Insights/types';
import { getOrderedQuerySnapshots } from '../queries';
import { MOCK_QUERY_SNAPSHOTS, MOCK_SAVED_QUERIES } from './mocks';

type ID = string;
type QueryRecord<T> = Record<ID, T>;

interface StoredQueriesContextValue {
  addUnsavedQuery: (query: UnsavedQuery) => void;
  deleteQuery: (queryId: string) => void;
  deleteQuerySnapshot: (snapshotId: string) => void;
  queries: QueryRecord<Query>;
  querySnapshots: QueryRecord<QuerySnapshot>;
  removeUnsavedQuery: (id: ID) => void;
  saveQuery: (query: Query, onSuccess: () => void) => void;
  saveQuerySnapshot: (snapshot: QuerySnapshot) => void;
}

const StoredQueriesContext = createContext<undefined | StoredQueriesContextValue>(undefined);

interface StoredQueriesProviderProps {
  children: ReactNode;
  tabManagerActions: TabManagerActions;
}

export function StoredQueriesProvider({ children, tabManagerActions }: StoredQueriesProviderProps) {
  const [querySnapshots, setQuerySnapshots] =
    useState<QueryRecord<QuerySnapshot>>(MOCK_QUERY_SNAPSHOTS);

  const [savedQueries = {}, setSavedQueries] = useLocalStorage<QueryRecord<Query>>(
    'insights-saved-queries',
    MOCK_SAVED_QUERIES
  );

  const [unsavedQueries, setUnsavedQueries] = useState<QueryRecord<UnsavedQuery>>({});

  const addUnsavedQuery = useCallback((query: UnsavedQuery) => {
    const unsavedQuery: UnsavedQuery = { ...query, saved: false };
    setUnsavedQueries((prev) => withId(prev, query.id, unsavedQuery));
  }, []);

  const removeUnsavedQuery = useCallback((id: ID) => {
    setUnsavedQueries((prev) => withoutId(prev, id));
  }, []);

  const saveQuery = useCallback(
    (query: Query, onSuccess: () => void) => {
      setSavedQueries(withId(savedQueries, query.id, { ...query, saved: true } as Query));
      setUnsavedQueries((prev) => withoutId(prev, query.id));
      onSuccess();
    },
    [setSavedQueries, savedQueries]
  );

  const deleteQuery = useCallback(
    (queryId: string) => {
      setSavedQueries(withoutId(savedQueries, queryId));
      tabManagerActions.breakQueryAssociation(queryId);
    },
    [savedQueries, setSavedQueries, tabManagerActions]
  );

  const deleteQuerySnapshot = useCallback(
    (snapshotId: string) => {
      setQuerySnapshots(withoutId(querySnapshots, snapshotId));
    },
    [querySnapshots, setQuerySnapshots]
  );

  const saveQuerySnapshot = useCallback(
    (snapshot: QuerySnapshot) => {
      setQuerySnapshots(
        withId(removeQuerySnapshotIfOverLimit(querySnapshots, 10), snapshot.id, snapshot)
      );
    },
    [setQuerySnapshots, querySnapshots]
  );

  const queries = useMemo(() => {
    return mergeRight(unsavedQueries, savedQueries);
  }, [unsavedQueries, savedQueries]);

  return (
    <StoredQueriesContext.Provider
      value={{
        addUnsavedQuery,
        deleteQuery,
        deleteQuerySnapshot,
        queries,
        querySnapshots,
        removeUnsavedQuery,
        saveQuery,
        saveQuerySnapshot,
      }}
    >
      {children}
    </StoredQueriesContext.Provider>
  );
}

export function useStoredQueries(): StoredQueriesContextValue {
  const context = useContext(StoredQueriesContext);
  if (context === undefined) {
    throw new Error('useStoredQueries must be used within a StoredQueriesProvider');
  }

  return context;
}

function mergeRight<T>(a: Record<string, T>, b: Record<string, T>): Record<string, T> {
  return { ...a, ...b };
}

function withId<T>(obj: Record<string, T>, id: string, value: T): Record<string, T> {
  return { ...obj, [id]: value };
}

function withoutId<T>(obj: Record<string, T>, id: string): Record<string, T> {
  const newObj = { ...obj };
  delete newObj[id];
  return newObj;
}

export function removeQuerySnapshotIfOverLimit(
  querySnapshots: QueryRecord<QuerySnapshot>,
  limit: number
): QueryRecord<QuerySnapshot> {
  if (Object.keys(querySnapshots).length < limit) return querySnapshots;

  const snapshots = getOrderedQuerySnapshots(querySnapshots);
  const oldestSnapshot = snapshots[snapshots.length - 1];
  if (oldestSnapshot === undefined) return querySnapshots;

  return withoutId(querySnapshots, oldestSnapshot.id);
}
