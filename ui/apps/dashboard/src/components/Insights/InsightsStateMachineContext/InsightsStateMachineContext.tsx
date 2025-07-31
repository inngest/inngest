'use client';

import { createContext, useCallback, useContext, useReducer, type ReactNode } from 'react';

import { simulateQuery } from './mocks';
import { insightsStateMachineReducer } from './reducer';
import type { InsightsState } from './types';

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

const INITIAL_STATE: InsightsState = {
  data: undefined,
  error: undefined,
  fetchMoreError: undefined,
  lastSentQuery: '',
  query: DEFAULT_QUERY,
  status: 'initial',
};

interface InsightsQueryContextValue extends InsightsState {
  fetchMore: () => void;
  isEmpty: boolean;
  onChange: (value: string) => void;
  runQuery: () => void;
}

const InsightsQueryContext = createContext<InsightsQueryContextValue | null>(null);

export function InsightsStateMachineContextProvider({ children }: { children: ReactNode }) {
  const [queryState, dispatch] = useReducer(insightsStateMachineReducer, INITIAL_STATE);

  const runQuery = useCallback(async () => {
    dispatch({ type: 'START_QUERY' });

    try {
      const result = await simulateQuery(queryState.query, null);
      dispatch({ type: 'QUERY_SUCCESS', payload: result });
    } catch (error) {
      dispatch({
        type: 'QUERY_ERROR',
        payload: stringifyError(error),
      });
    }
  }, [queryState.query]);

  const onChange = useCallback((value: string) => {
    dispatch({ type: 'UPDATE_CONTENT', payload: value });
  }, []);

  const fetchMore = useCallback(async () => {
    dispatch({ type: 'FETCH_MORE' });

    try {
      const result = await simulateQuery(
        queryState.lastSentQuery,
        queryState.data?.pageInfo.endCursor ?? null
      );
      dispatch({ type: 'FETCH_MORE_SUCCESS', payload: result });
    } catch (error) {
      dispatch({
        type: 'FETCH_MORE_ERROR',
        payload: stringifyError(error),
      });
    }
  }, [queryState.data?.pageInfo.endCursor, queryState.lastSentQuery]);

  return (
    <InsightsQueryContext.Provider
      value={{
        ...queryState,
        fetchMore,
        isEmpty: queryState.query.trim() === '',
        onChange,
        runQuery,
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

export function useInsightsStateMachineContext() {
  const context = useContext(InsightsQueryContext);
  if (!context) {
    throw new Error(
      'useInsightsStateMachineContext must be used within InsightsStateMachineContextProvider'
    );
  }

  return context;
}
