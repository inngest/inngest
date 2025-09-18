'use client';

import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';
import { toast } from 'sonner';

import type { TabManagerActions } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import type { Query, QuerySnapshot, Tab } from '@/components/Insights/types';
import { getOrderedQuerySnapshots, getOrderedSavedQueries } from '../queries';
import { useInsightsSavedQueries } from './useInsightsSavedQueries';

type ID = string;
type QueryRecord<T> = Record<ID, T>;

interface StoredQueriesContextValue {
  deleteQuery: (queryId: string) => void;
  deleteQuerySnapshot: (snapshotId: string) => void;
  isSavedQueriesFetching: boolean;
  queries: {
    data: undefined | Query[];
    error: undefined | string;
    isLoading: boolean;
  };
  querySnapshots: {
    data: QuerySnapshot[];
    error: undefined;
    isLoading: boolean;
  };
  saveQuery: (tab: Tab) => Promise<void>;
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
    deleteQuery: beDeleteQuery,
    savedQueries: beSavedQueries,
    savedQueriesError,
    isSavedQueriesFetching,
    saveQuery: beSaveQuery,
    updateQuery: beUpdateQuery,
    refetchSavedQueries,
  } = useInsightsSavedQueries();

  const saveQuery = useCallback(
    async (tab: Tab) => {
      if (tab.savedQueryId !== undefined) {
        try {
          await beUpdateQuery({ id: tab.savedQueryId, name: tab.name, query: tab.query });
          toast.success('Successfully updated query');
        } catch (e) {
          toast.error('Failed to update query');
        }
      } else {
        try {
          const saved = await beSaveQuery({ name: tab.name, query: tab.query });
          tabManagerActions.updateTab(tab.id, { savedQueryId: saved.id });
          toast.success('Successfully saved query');
        } catch (e) {
          toast.error('Failed to save query');
        }
      }
    },
    [beSaveQuery, beUpdateQuery, tabManagerActions]
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
    [beDeleteQuery, refetchSavedQueries, tabManagerActions]
  );

  const deleteQuerySnapshot = useCallback(
    (snapshotId: string) => {
      setQuerySnapshots(withoutId(querySnapshots, snapshotId));
    },
    [querySnapshots, setQuerySnapshots]
  );

  const saveQuerySnapshot = useCallback(
    (snapshot: QuerySnapshot) => {
      setQuerySnapshots((current) =>
        withId(removeQuerySnapshotIfOverLimit(current, 10), snapshot.id, snapshot)
      );
    },
    [setQuerySnapshots]
  );

  const queries = useMemo(() => {
    return {
      data: getOrderedSavedQueries(beSavedQueries),
      error: savedQueriesError ? savedQueriesError.message : undefined,
      isLoading: isSavedQueriesFetching,
    };
  }, [beSavedQueries, isSavedQueriesFetching, savedQueriesError]);

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
        deleteQuery,
        deleteQuerySnapshot,
        isSavedQueriesFetching,
        queries,
        querySnapshots: orderedQuerySnapshots,
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
