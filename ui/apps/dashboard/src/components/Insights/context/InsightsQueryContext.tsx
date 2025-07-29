'use client';

import { createContext, useCallback, useContext, useReducer, type ReactNode } from 'react';

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

const MOCK_DATA: InsightsResult = {
  columns: ['hour_bucket', 'Column name'],
  rows: [
    {
      hour_bucket: '11/22/2023 1:30:01 PM',
      'Column name': 'brand/manual-add-off-platform-brands-created',
    },
    { hour_bucket: '11/22/2023 1:30:01 PM', 'Column name': 'action-created' },
    { hour_bucket: '11/22/2023 1:30:01 PM', 'Column name': 'clerk/organisation/MembershipDeleted' },
  ],
  totalRows: 3,
};

function simulateQuery(): Promise<InsightsResult> {
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      const rand = Math.random();
      if (rand < 0.7) {
        resolve(MOCK_DATA);
      } else if (rand < 0.9) {
        resolve({ columns: ['hour', 'count'], rows: [], totalRows: 0 });
      } else {
        reject(new Error('Query execution failed. Please check your syntax and try again.'));
      }
    }, 2500);
  });
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
  | { type: 'QUERY_ERROR'; payload: string };

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
      const result = await simulateQuery();
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

  return (
    <InsightsQueryContext.Provider
      value={{
        content: queryState.content,
        data: queryState.data,
        error: queryState.error,
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
