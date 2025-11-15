'use client';

import { useCallback, useMemo } from 'react';
import { useQuery, type CombinedError } from 'urql';

import { graphql } from '@/gql';
import type { InsightsQueryStatement } from '@/gql/graphql';
import {
  useModifySavedQueries,
  type DeleteQueryArgs,
  type MutationResult,
  type SaveQueryArgs,
  type ShareQueryArgs,
  type UpdateQueryArgs,
} from './useModifySavedQueries';

const insightsSavedQueriesQuery = graphql(`
  query InsightsSavedQueries {
    account {
      insightsQueries {
        id
        creator
        lastEditor
        name
        shared
        sql
        createdAt
        updatedAt
      }
    }
  }
`);

export interface UseInsightsSavedQueriesReturn {
  deleteQuery: (args: DeleteQueryArgs) => Promise<MutationResult<string[]>>;
  isSavedQueriesFetching: boolean;
  refetchSavedQueries: () => void;
  savedQueries: undefined | InsightsQueryStatement[];
  savedQueriesError: undefined | CombinedError;
  saveQuery: (args: SaveQueryArgs) => Promise<MutationResult<InsightsQueryStatement>>;
  shareQuery: (args: ShareQueryArgs) => Promise<MutationResult<InsightsQueryStatement>>;
  updateQuery: (args: UpdateQueryArgs) => Promise<MutationResult<InsightsQueryStatement>>;
}

export function useInsightsSavedQueries(): UseInsightsSavedQueriesReturn {
  const [result, reexecute] = useQuery({ query: insightsSavedQueriesQuery });
  const { deleteQuery, saveQuery, shareQuery, updateQuery } = useModifySavedQueries();

  const refetchSavedQueries = useCallback(() => {
    reexecute({ requestPolicy: 'network-only' });
  }, [reexecute]);

  const savedQueries = useMemo(() => result.data?.account.insightsQueries, [result.data]);

  return {
    deleteQuery,
    isSavedQueriesFetching: result.fetching,
    refetchSavedQueries,
    savedQueries,
    savedQueriesError: result.error,
    saveQuery,
    shareQuery,
    updateQuery,
  };
}
