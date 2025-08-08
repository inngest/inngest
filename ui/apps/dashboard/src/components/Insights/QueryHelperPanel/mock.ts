import { useEffect, useState } from 'react';

import type { Query } from './types';

const MOCK_TEMPLATES: Query[] = [
  {
    id: 'template-1',
    name: 'Template 1',
    query: 'Template 1 Query Text',
    type: 'template',
  },
  {
    id: 'template-2',
    name: 'Template 2',
    query: 'Template 2 Query Text',
    type: 'template',
  },
];

const MOCK_RECENT_QUERIES: Query[] = [
  {
    id: 'recent-query-1',
    name: 'SELECT COUNT(*) FROM events WHERE ts > NOW() - INTERVAL 1 HOUR',
    query: 'SELECT COUNT(*) FROM events WHERE ts > NOW() - INTERVAL 1 HOUR',
    type: 'recent',
  },
  {
    id: 'recent-query-2',
    name: 'SELECT status, COUNT(*) FROM function_runs GROUP BY status',
    query: 'SELECT status, COUNT(*) FROM function_runs GROUP BY status',
    type: 'recent',
  },
];

const MOCK_SAVED_QUERIES: Query[] = [
  {
    id: 'saved-query-1',
    name: 'Saved Query 1',
    query: 'Saved Query 1 Query Text',
    type: 'saved',
  },
  {
    id: 'saved-query-2',
    name: 'Saved Query 2',
    query: 'Saved Query 2 Query Text',
    type: 'saved',
  },
];

interface QueryResult<T> {
  data: T | undefined;
  error: string | undefined;
  isLoading: boolean;
}

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

export function useTemplates(): QueryResult<Query[]> {
  return { data: MOCK_TEMPLATES, error: undefined, isLoading: false };
}

export function useRecentQueries(): QueryResult<Query[]> {
  const [state, setState] = useState<QueryResult<Query[]>>({
    data: undefined,
    error: undefined,
    isLoading: true,
  });

  useEffect(() => {
    simulateAsyncLoad(MOCK_RECENT_QUERIES, 1000, setState);
  }, []);

  return state;
}

export function useSavedQueries(): QueryResult<Query[]> {
  const [state, setState] = useState<QueryResult<Query[]>>({
    data: undefined,
    error: undefined,
    isLoading: true,
  });

  useEffect(() => {
    simulateAsyncLoad(MOCK_SAVED_QUERIES, 500, setState);
  }, []);

  return state;
}
