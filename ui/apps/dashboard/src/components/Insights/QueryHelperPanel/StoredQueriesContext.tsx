'use client';

import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';
import { toast } from 'sonner';

import type { TabManagerActions } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import type { QuerySnapshot, Tab } from '@/components/Insights/types';
import type { InsightsQuery } from '@/gql/graphql';
import { getOrderedSavedQueries } from '../queries';
import { useInsightsSavedQueries } from './useInsightsSavedQueries';

interface StoredQueriesContextValue {
  deleteQuery: (queryId: string) => void;
  deleteQuerySnapshot: (snapshotId: string) => void;
  isSavedQueriesFetching: boolean;
  queries: {
    data: undefined | InsightsQuery[];
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
  const [querySnapshots, setQuerySnapshots] = useState<QuerySnapshot[]>([]);

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

  const deleteQuerySnapshot = useCallback((snapshotId: string) => {
    setQuerySnapshots((prev) => prev.filter((s) => s.id !== snapshotId));
  }, []);

  const saveQuerySnapshot = useCallback((snapshot: QuerySnapshot) => {
    setQuerySnapshots((current) => [snapshot, ...current].slice(0, 10));
  }, []);

  const queries = useMemo(() => {
    return {
      data: getOrderedSavedQueries(beSavedQueries),
      error: savedQueriesError ? savedQueriesError.message : undefined,
      isLoading: isSavedQueriesFetching,
    };
  }, [beSavedQueries, isSavedQueriesFetching, savedQueriesError]);

  const orderedQuerySnapshots = useMemo(
    () => ({ data: querySnapshots, error: undefined, isLoading: false }),
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
