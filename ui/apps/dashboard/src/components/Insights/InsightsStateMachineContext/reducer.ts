import type { InsightsAction, InsightsState } from './types';

export function insightsStateMachineReducer(
  state: InsightsState,
  action: InsightsAction
): InsightsState {
  switch (action.type) {
    case 'FETCH_MORE':
      return { ...state, fetchMoreError: undefined, status: 'fetchingMore' };

    case 'FETCH_MORE_ERROR':
      return { ...state, fetchMoreError: action.payload, status: 'fetchMoreError' };

    case 'FETCH_MORE_SUCCESS':
      return {
        ...state,
        data: state.data
          ? { ...action.payload, entries: [...state.data.entries, ...action.payload.entries] }
          : action.payload,
        fetchMoreError: undefined,
        status: 'success',
      };

    case 'QUERY_ERROR':
      return {
        ...state,
        data: undefined,
        error: action.payload,
        fetchMoreError: undefined,
        status: 'error',
      };

    case 'QUERY_SUCCESS':
      return {
        ...state,
        data: action.payload,
        error: undefined,
        fetchMoreError: undefined,
        status: 'success',
      };

    case 'START_QUERY':
      return {
        ...state,
        data: undefined,
        error: undefined,
        fetchMoreError: undefined,
        lastSentQuery: action.payload,
        status: 'loading',
      };

    default:
      return state;
  }
}
