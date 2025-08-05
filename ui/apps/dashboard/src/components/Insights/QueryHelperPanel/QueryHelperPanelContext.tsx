'use client';

import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';
import { useLocalStorage } from 'react-use';

import { TEMPLATE_QUERIES } from './templates';
import type { RecentQuery, SavedQuery } from './types';

export type UseQueryHelperPanelReturn = {
  addRecentQuery: (query: Omit<RecentQuery, 'updatedOn'>) => void;
  addSavedQuery: (query: Omit<SavedQuery, 'updatedOn'>) => void;
  recentQueries: RecentQuery[];
  removeRecentQuery: (id: string) => void;
  removeSavedQuery: (id: string) => void;
  savedQueries: SavedQuery[];
  templates: SavedQuery[];
};

const QueryHelperPanelContext = createContext<UseQueryHelperPanelReturn | null>(null);

export function QueryHelperPanelProvider({ children }: { children: ReactNode }) {
  const [recentQueriesRaw, setRecentQueries] = useLocalStorage(
    'insightsRecentQueries',
    [] as RecentQuery[]
  );
  const [savedQueriesRaw, setSavedQueries] = useLocalStorage(
    'insightsSavedQueries',
    [] as SavedQuery[]
  );
  const [templates] = useState<SavedQuery[]>(TEMPLATE_QUERIES);

  const recentQueries = useMemo(() => sortByUpdatedOn(recentQueriesRaw ?? []), [recentQueriesRaw]);

  const savedQueries = useMemo(() => sortByUpdatedOn(savedQueriesRaw ?? []), [savedQueriesRaw]);

  const addRecentQuery = useCallback(
    (query: Omit<RecentQuery, 'updatedOn'>) => {
      setRecentQueries(addQueryWithTimestamp(query, recentQueries));
    },
    [recentQueries, setRecentQueries]
  );

  const addSavedQuery = useCallback(
    (query: Omit<SavedQuery, 'updatedOn'>) => {
      setSavedQueries(addQueryWithTimestamp(query, savedQueries));
    },
    [savedQueries, setSavedQueries]
  );

  const removeRecentQuery = useCallback(
    (id: string) => {
      setRecentQueries(recentQueries.filter((q) => q.id !== id));
    },
    [recentQueries, setRecentQueries]
  );

  const removeSavedQuery = useCallback(
    (id: string) => {
      setSavedQueries(savedQueries.filter((q) => q.id !== id));
    },
    [savedQueries, setSavedQueries]
  );

  return (
    <QueryHelperPanelContext.Provider
      value={{
        addRecentQuery,
        addSavedQuery,
        recentQueries,
        removeRecentQuery,
        removeSavedQuery,
        savedQueries,
        templates,
      }}
    >
      {children}
    </QueryHelperPanelContext.Provider>
  );
}

export function useQueryHelperPanelContext() {
  const context = useContext(QueryHelperPanelContext);
  if (!context) {
    throw new Error('useQueryHelperPanelContext must be used within QueryHelperPanelProvider');
  }

  return context;
}

function sortByUpdatedOn<T extends { updatedOn: string }>(queries: T[]): T[] {
  return [...queries].sort((a, b) => b.updatedOn.localeCompare(a.updatedOn));
}

function addQueryWithTimestamp<T extends { id: string; updatedOn: string }>(
  query: Omit<T, 'updatedOn'>,
  existingQueries: T[]
): T[] {
  const queryWithTimestamp = { ...query, updatedOn: new Date().toISOString() } as T;
  return sortByUpdatedOn([queryWithTimestamp, ...existingQueries]);
}
