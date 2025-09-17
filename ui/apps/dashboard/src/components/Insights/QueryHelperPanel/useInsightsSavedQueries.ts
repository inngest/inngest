'use client';

import { useCallback, useMemo } from 'react';
import { useQuery, type CombinedError } from 'urql';

import { graphql } from '@/gql';
import { toLocalQueryArray } from '../queries';
import type { Query as InsightsQueryLocal } from '../types';
import {
  useModifySavedQueries,
  type DeleteQueryArgs,
  type SaveQueryArgs,
  type UpdateQueryArgs,
} from './useModifySavedQueries';

const insightsSavedQueriesQuery = graphql(`
  query InsightsSavedQueries {
    account {
      insightsQueries {
        id
        name
        sql
        createdAt
        updatedAt
      }
    }
  }
`);

export interface UseInsightsSavedQueriesReturn {
  deleteQuery: (args: DeleteQueryArgs) => Promise<string[]>;
  isSavedQueriesFetching: boolean;
  refetchSavedQueries: () => void;
  savedQueries: undefined | InsightsQueryLocal[];
  savedQueriesError: undefined | CombinedError;
  saveQuery: (args: SaveQueryArgs) => Promise<InsightsQueryLocal>;
  updateQuery: (args: UpdateQueryArgs) => Promise<InsightsQueryLocal>;
}

export function useInsightsSavedQueries(): UseInsightsSavedQueriesReturn {
  const [result, reexecute] = useQuery({ query: insightsSavedQueriesQuery });
  const { saveQuery, updateQuery, deleteQuery } = useModifySavedQueries();

  const refetchSavedQueries = useCallback(() => {
    reexecute({ requestPolicy: 'network-only' });
  }, [reexecute]);

  const savedQueries = useMemo(
    () => toLocalQueryArray(result.data?.account.insightsQueries),
    [result.data]
  );

  return {
    deleteQuery,
    isSavedQueriesFetching: result.fetching,
    refetchSavedQueries,
    savedQueries,
    savedQueriesError: result.error,
    saveQuery,
    updateQuery,
  };
}
