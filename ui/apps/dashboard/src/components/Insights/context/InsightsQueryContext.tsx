'use client';

import { createContext, useCallback, useContext, useReducer, type ReactNode } from 'react';

import { getMockPage } from '../InsightsDataTable/mocks';
import { type InsightsResult, type InsightsState } from '../InsightsDataTable/types';

const DEFAULT_QUERY = `SELECT 
  HOUR(ts) as hour, 
  COUNT(*) as count 
WHERE 
  name = 'cli/dev_ui.loaded' 
  AND data.os != 'linux'
  AND ts > 1752845983000 
GROUP BY
  hour 
ORDER BY 
  hour desc`;

async function simulateQuery(cursor: string | null): Promise<InsightsResult> {
  await new Promise((resolve) => setTimeout(resolve, 3000 + Math.random() * 1000));

  if (Math.random() < 0.05) {
    throw new Error('Query timeout - please try a more specific query');
  }

  return getMockPage(cursor);
}

interface InsightsQueryState {
  content: string;
  data?: InsightsResult;
  error?: string;
  fetchMoreError?: string;
  state: InsightsState;
}

type InsightsQueryAction =
  | { type: 'UPDATE_CONTENT'; payload: string }
  | { type: 'START_QUERY' }
  | { type: 'QUERY_SUCCESS'; payload: InsightsResult }
  | { type: 'QUERY_ERROR'; payload: string }
  | { type: 'FETCH_MORE' }
  | { type: 'FETCH_MORE_SUCCESS'; payload: InsightsResult }
  | { type: 'FETCH_MORE_ERROR'; payload: string };

function insightsQueryReducer(
  state: InsightsQueryState,
  action: InsightsQueryAction
): InsightsQueryState {
  switch (action.type) {
    case 'UPDATE_CONTENT':
      return { ...state, content: action.payload };
    case 'START_QUERY':
      return {
        ...state,
        data: undefined,
        error: undefined,
        fetchMoreError: undefined,
        state: 'loading',
      };
    case 'QUERY_SUCCESS':
      return {
        ...state,
        data: action.payload,
        error: undefined,
        fetchMoreError: undefined,
        state: 'success',
      };
    case 'QUERY_ERROR':
      return {
        ...state,
        data: undefined,
        error: action.payload,
        fetchMoreError: undefined,
        state: 'error',
      };
    case 'FETCH_MORE':
      return { ...state, fetchMoreError: undefined, state: 'fetchingMore' };
    case 'FETCH_MORE_SUCCESS':
      return {
        ...state,
        data: state.data
          ? {
              ...action.payload,
              entries: [...state.data.entries, ...action.payload.entries],
            }
          : action.payload,
        fetchMoreError: undefined,
        state: 'success',
      };
    case 'FETCH_MORE_ERROR':
      return { ...state, fetchMoreError: action.payload, state: 'fetchMoreError' };
    default:
      return state;
  }
}

const initialState: InsightsQueryState = {
  content: DEFAULT_QUERY,
  state: 'initial',
};

interface InsightsQueryContextValue {
  content: string;
  data?: InsightsResult;
  error?: string;
  fetchMore: () => void;
  fetchMoreError?: string;
  isEmpty: boolean;
  onChange: (value: string) => void;
  runQuery: () => void;
  state: InsightsState;
}

const InsightsQueryContext = createContext<InsightsQueryContextValue | null>(null);

export function InsightsQueryContextProvider({ children }: { children: ReactNode }) {
  const [queryState, dispatch] = useReducer(insightsQueryReducer, initialState);

  const runQuery = useCallback(async () => {
    dispatch({ type: 'START_QUERY' });

    try {
      const result = await simulateQuery(null);
      dispatch({ type: 'QUERY_SUCCESS', payload: result });
    } catch (error) {
      dispatch({
        type: 'QUERY_ERROR',
        payload: stringifyError(error),
      });
    }
  }, [queryState.state]);

  const onChange = useCallback((value: string) => {
    dispatch({ type: 'UPDATE_CONTENT', payload: value });
  }, []);

  const fetchMore = useCallback(async () => {
    dispatch({ type: 'FETCH_MORE' });

    try {
      const result = await simulateQuery(queryState.data?.pageInfo.endCursor ?? null);
      dispatch({ type: 'FETCH_MORE_SUCCESS', payload: result });
    } catch (error) {
      dispatch({
        type: 'FETCH_MORE_ERROR',
        payload: stringifyError(error),
      });
    }
  }, [
    queryState.state,
    queryState.data?.pageInfo.hasNextPage,
    queryState.data?.pageInfo.endCursor,
  ]);

  return (
    <InsightsQueryContext.Provider
      value={{
        content: queryState.content,
        data: queryState.data,
        error: queryState.error,
        fetchMore,
        fetchMoreError: queryState.fetchMoreError,
        isEmpty: queryState.content.trim() === '',
        onChange,
        runQuery,
        state: queryState.state,
      }}
    >
      {children}
    </InsightsQueryContext.Provider>
  );
}

function stringifyError(error: unknown): string {
  if (error instanceof Error) return error.message;
  return 'Unknown error';
}

export function useInsightsQueryContext() {
  const context = useContext(InsightsQueryContext);
  if (!context)
    throw new Error('useInsightsQueryContext must be used within InsightsQueryContextProvider');

  return context;
}
