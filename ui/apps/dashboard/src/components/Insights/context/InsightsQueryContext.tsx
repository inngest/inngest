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
  await new Promise((resolve) => setTimeout(resolve, 500 + Math.random() * 1000));

  if (Math.random() < 0.05) {
    throw new Error('Query timeout - please try a more specific query');
  }

  return getMockPage(cursor);
}

interface InsightsQueryState {
  content: string;
  data?: InsightsResult;
  error?: string;
  state: InsightsState;
}

type InsightsQueryAction =
  | { type: 'UPDATE_CONTENT'; payload: string }
  | { type: 'START_QUERY' }
  | { type: 'QUERY_SUCCESS'; payload: InsightsResult }
  | { type: 'QUERY_ERROR'; payload: string }
  | { type: 'FETCH_MORE' }
  | { type: 'FETCH_MORE_SUCCESS'; payload: InsightsResult };

function insightsQueryReducer(
  state: InsightsQueryState,
  action: InsightsQueryAction
): InsightsQueryState {
  switch (action.type) {
    case 'UPDATE_CONTENT':
      return { ...state, content: action.payload };
    case 'START_QUERY':
      return { ...state, data: undefined, error: undefined, state: 'loading' };
    case 'QUERY_SUCCESS':
      return { ...state, data: action.payload, state: 'success' };
    case 'QUERY_ERROR':
      return { ...state, data: undefined, error: action.payload, state: 'error' };
    case 'FETCH_MORE':
      return { ...state, state: 'fetchingMore' };
    case 'FETCH_MORE_SUCCESS':
      return {
        ...state,
        data: state.data
          ? {
              ...action.payload,
              entries: [...state.data.entries, ...action.payload.entries],
            }
          : action.payload,
        state: 'success',
      };
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
  isEmpty: boolean;
  onChange: (value: string) => void;
  runQuery: () => void;
  seeExamples: () => void;
  state: InsightsState;
}

const InsightsQueryContext = createContext<InsightsQueryContextValue | null>(null);

export function InsightsQueryContextProvider({ children }: { children: ReactNode }) {
  const [queryState, dispatch] = useReducer(insightsQueryReducer, initialState);

  const runQuery = useCallback(async () => {
    if (queryState.state === 'loading') return;

    dispatch({ type: 'START_QUERY' });

    try {
      const result = await simulateQuery(null); // Initial query starts with null cursor
      dispatch({ type: 'QUERY_SUCCESS', payload: result });
    } catch (error) {
      dispatch({
        type: 'QUERY_ERROR',
        payload: error instanceof Error ? error.message : 'Unknown error',
      });
    }
  }, [queryState.state]);

  const onChange = useCallback((value: string) => {
    dispatch({ type: 'UPDATE_CONTENT', payload: value });
  }, []);

  const seeExamples = useCallback(() => console.log('TODO: Show examples'), []);

  const fetchMore = useCallback(async () => {
    if (queryState.state !== 'success' || !queryState.data?.pageInfo.hasNextPage) return;

    dispatch({ type: 'FETCH_MORE' });

    try {
      const result = await simulateQuery(queryState.data?.pageInfo.endCursor);
      dispatch({ type: 'FETCH_MORE_SUCCESS', payload: result });
    } catch (error) {
      dispatch({
        type: 'QUERY_ERROR',
        payload: error instanceof Error ? error.message : 'Unknown error',
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
        isEmpty: queryState.content.trim() === '',
        onChange,
        runQuery,
        seeExamples,
        state: queryState.state,
      }}
    >
      {children}
    </InsightsQueryContext.Provider>
  );
}

export function useInsightsQueryContext() {
  const context = useContext(InsightsQueryContext);
  if (!context)
    throw new Error('useInsightsQueryContext must be used within InsightsQueryContextProvider');

  return context;
}
