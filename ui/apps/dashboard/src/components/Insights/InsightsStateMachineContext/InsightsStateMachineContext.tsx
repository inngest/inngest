'use client';

import { createContext, useCallback, useContext, useReducer, type ReactNode } from 'react';

import { useStoredQueries } from '../QueryHelperPanel/StoredQueriesContext';
import { makeQuerySnapshot } from '../queries';
import { simulateQuery } from './mocks';
import { insightsStateMachineReducer } from './reducer';
import type { InsightsState } from './types';

const INITIAL_STATE: InsightsState = {
  data: undefined,
  error: undefined,
  fetchMoreError: undefined,
  lastSentQuery: '',
  status: 'initial',
};

interface InsightsStateMachineContextValue extends InsightsState {
  fetchMore: () => void;
  isEmpty: boolean;
  onChange: (value: string) => void;
  onNameChange: (name: string) => void;
  query: string;
  queryName: string;
  runQuery: () => void;
}

const InsightsStateMachineContext = createContext<InsightsStateMachineContextValue | null>(null);

type InsightsStateMachineContextProviderProps = {
  children: ReactNode;
  onQueryChange: (query: string) => void;
  onQueryNameChange: (name: string) => void;
  query: string;
  queryName: string;
  renderChildren: boolean;
};

export function InsightsStateMachineContextProvider({
  children,
  onQueryChange,
  onQueryNameChange,
  query,
  queryName,
  renderChildren,
}: InsightsStateMachineContextProviderProps) {
  const [queryState, dispatch] = useReducer(insightsStateMachineReducer, INITIAL_STATE);
  const { saveQuerySnapshot } = useStoredQueries();

  // TODO: Ensure runQuery and fetchMore cannot finish out of order
  // if run in quick succession.
  const runQuery = useCallback(async () => {
    dispatch({ type: 'START_QUERY', payload: query });

    try {
      const result = await simulateQuery(query, null);
      dispatch({ type: 'QUERY_SUCCESS', payload: result });
      saveQuerySnapshot(makeQuerySnapshot(query));
    } catch (error) {
      dispatch({
        type: 'QUERY_ERROR',
        payload: stringifyError(error),
      });
    }
  }, [query, saveQuerySnapshot]);

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
    <InsightsStateMachineContext.Provider
      value={{
        ...queryState,
        query,
        queryName,
        fetchMore,
        isEmpty: query.trim() === '',
        onChange: onQueryChange,
        onNameChange: onQueryNameChange,
        runQuery,
      }}
    >
      {renderChildren ? children : null}
    </InsightsStateMachineContext.Provider>
  );
}

function stringifyError(error: unknown): string {
  if (error instanceof Error) return error.message;
  return 'Unknown error';
}

export function useInsightsStateMachineContext() {
  const context = useContext(InsightsStateMachineContext);
  if (!context) {
    throw new Error(
      'useInsightsStateMachineContext must be used within InsightsStateMachineContextProvider'
    );
  }

  return context;
}
