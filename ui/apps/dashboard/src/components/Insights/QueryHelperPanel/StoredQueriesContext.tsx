'use client';

import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';
import { toast } from 'sonner';

import type { TabManagerActions } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import type { Query, QuerySnapshot, UnsavedQuery } from '@/components/Insights/types';
import { getOrderedQuerySnapshots, getOrderedSavedQueries } from '../queries';
import { useInsightsSavedQueries } from './useInsightsSavedQueries';

type ID = string;
type QueryRecord<T> = Record<ID, T>;

interface StoredQueriesContextValue {
  addUnsavedQuery: (query: UnsavedQuery) => void;
  deleteQuery: (queryId: string) => void;
  deleteQuerySnapshot: (snapshotId: string) => void;
  isSavedQueriesFetching: boolean;
  queries: QueryRecord<Query>;
  querySnapshots: {
    data: QuerySnapshot[];
    error: undefined;
    isLoading: boolean;
  };
  removeUnsavedQuery: (id: ID) => void;
  savedQueries: {
    data: undefined | Query[];
    error: undefined | string;
    isLoading: boolean;
  };
  savedQueriesError: undefined | string;
  saveQuery: (query: Query, onSuccess: () => void) => void;
  updateQuery: (query: Query, onSuccess: () => void) => void;
  saveQuerySnapshot: (snapshot: QuerySnapshot) => void;
}

const StoredQueriesContext = createContext<undefined | StoredQueriesContextValue>(undefined);

interface StoredQueriesProviderProps {
  children: ReactNode;
  tabManagerActions: TabManagerActions;
}

export function StoredQueriesProvider({ children, tabManagerActions }: StoredQueriesProviderProps) {
  const [querySnapshots, setQuerySnapshots] = useState<QueryRecord<QuerySnapshot>>({});

  const {
    savedQueries: beSavedQueries,
    savedQueriesError,
    isSavedQueriesFetching,
    saveQuery: beSaveQuery,
    updateQuery: beUpdateQuery,
    deleteQuery: beDeleteQuery,
    refetchSavedQueries,
  } = useInsightsSavedQueries();

  const [unsavedQueries, setUnsavedQueries] = useState<QueryRecord<UnsavedQuery>>({});

  const addUnsavedQuery = useCallback((query: UnsavedQuery) => {
    const unsavedQuery: UnsavedQuery = { ...query, saved: false };
    setUnsavedQueries((prev) => withId(prev, query.id, unsavedQuery));
  }, []);

  const removeUnsavedQuery = useCallback((id: ID) => {
    setUnsavedQueries((prev) => withoutId(prev, id));
  }, []);

  const saveQuery = useCallback(
    async (query: Query, onSuccess: () => void) => {
      try {
        await beSaveQuery({ name: query.name, query: query.query });
        setUnsavedQueries((prev) => withoutId(prev, query.id));
        onSuccess();
        refetchSavedQueries();
        toast.success('Query created');
      } catch (e) {
        toast.error('Failed to create query');
      }
    },
    [beSaveQuery, refetchSavedQueries]
  );

  const deleteQuery = useCallback(
    async (queryId: string) => {
      try {
        await beDeleteQuery({ id: queryId });
        tabManagerActions.breakQueryAssociation(queryId);
        refetchSavedQueries();
        toast.success('Query deleted');
      } catch (e) {
        toast.error('Failed to delete query');
      }
    },
    [beDeleteQuery, tabManagerActions, refetchSavedQueries]
  );

  const updateQuery = useCallback(
    async (query: Query, onSuccess: () => void) => {
      try {
        await beUpdateQuery({ id: query.id, name: query.name, query: query.query });
        onSuccess();
        refetchSavedQueries();
        toast.success('Query updated');
      } catch (e) {
        toast.error('Failed to update query');
      }
    },
    [beUpdateQuery, refetchSavedQueries]
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
    const beQueries: QueryRecord<Query> = Object.fromEntries(
      (beSavedQueries ?? []).map((q) => [q.id, q])
    );
    return mergeRight(unsavedQueries, beQueries);
  }, [unsavedQueries, beSavedQueries]);

  const savedQueries = useMemo(() => {
    return {
      data: getOrderedSavedQueries(queries),
      error: savedQueriesError ? savedQueriesError.message : undefined,
      isLoading: isSavedQueriesFetching,
    };
  }, [queries, savedQueriesError, isSavedQueriesFetching]);

  const orderedQuerySnapshots = useMemo(
    () => ({
      data: getOrderedQuerySnapshots(querySnapshots),
      error: undefined,
      isLoading: false,
    }),
    [querySnapshots]
  );

  return (
    <StoredQueriesContext.Provider
      value={{
        addUnsavedQuery,
        deleteQuery,
        deleteQuerySnapshot,
        isSavedQueriesFetching,
        queries,
        querySnapshots: orderedQuerySnapshots,
        removeUnsavedQuery,
        savedQueries,
        savedQueriesError: savedQueriesError?.message,
        saveQuery,
        saveQuerySnapshot,
        updateQuery,
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
