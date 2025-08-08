'use client';

import { createContext, useContext, type ReactNode } from 'react';

import { useRecentQueries, useSavedQueries, useTemplates } from './mock';
import type { Query } from './types';

interface QueryHelperPanelContextValue {
  recentQueries: {
    data: undefined | Query[];
    error: undefined | string;
    isLoading: boolean;
  };
  savedQueries: {
    data: undefined | Query[];
    error: undefined | string;
    isLoading: boolean;
  };
  templates: {
    data: undefined | Query[];
    error: undefined | string;
    isLoading: boolean;
  };
}

const QueryHelperPanelContext = createContext<null | QueryHelperPanelContextValue>(null);

interface QueryHelperPanelContextProviderProps {
  children: ReactNode;
}

export function QueryHelperPanelContextProvider({
  children,
}: QueryHelperPanelContextProviderProps) {
  const recentQueries = useRecentQueries();
  const savedQueries = useSavedQueries();
  const templates = useTemplates();

  return (
    <QueryHelperPanelContext.Provider value={{ recentQueries, savedQueries, templates }}>
      {children}
    </QueryHelperPanelContext.Provider>
  );
}

export function useQueryHelperPanelContext() {
  const context = useContext(QueryHelperPanelContext);
  if (!context) {
    throw new Error(
      'useQueryHelperPanelContext must be used within QueryHelperPanelContextProvider'
    );
  }

  return context;
}
