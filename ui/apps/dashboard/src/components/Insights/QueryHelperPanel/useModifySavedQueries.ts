'use client';

import { useCallback } from 'react';
import { useMutation, type CombinedError } from 'urql';

import { graphql } from '@/gql';
import type {
  CreateInsightsQueryMutation,
  InsightsQuery,
  RemoveInsightsQueryMutation,
  UpdateInsightsQueryMutation,
} from '@/gql/graphql';

const createInsightsQueryDocument = graphql(`
  mutation CreateInsightsQuery($input: NewInsightsQuery!) {
    createInsightsQuery(input: $input) {
      id
      name
      sql
      createdAt
      updatedAt
    }
  }
`);

const removeInsightsQueryDocument = graphql(`
  mutation RemoveInsightsQuery($id: ID!) {
    removeInsightsQuery(id: $id) {
      ids
    }
  }
`);

const updateInsightsQueryDocument = graphql(`
  mutation UpdateInsightsQuery($id: ID!, $input: UpdateInsightsQuery!) {
    updateInsightsQuery(id: $id, input: $input) {
      id
      name
      sql
      createdAt
      updatedAt
    }
  }
`);

export type DeleteQueryArgs = { id: string };
export type SaveQueryArgs = { name: string; query: string };
export type UpdateQueryArgs = { id: string; name: string; query: string };

type UseModifySavedQueriesReturn = {
  deleteQuery: (args: DeleteQueryArgs) => Promise<string[]>;
  saveQuery: (args: SaveQueryArgs) => Promise<InsightsQuery>;
  updateQuery: (args: UpdateQueryArgs) => Promise<InsightsQuery>;
};

export function useModifySavedQueries(): UseModifySavedQueriesReturn {
  const [, runCreate] = useMutation(createInsightsQueryDocument);
  const [, runRemove] = useMutation(removeInsightsQueryDocument);
  const [, runUpdate] = useMutation(updateInsightsQueryDocument);

  const executeMutation = useCallback(
    async <T>(fn: () => Promise<{ data?: T; error?: CombinedError }>): Promise<T> => {
      const res = await fn();
      if (res.error) throw res.error;
      if (res.data === undefined) throw new Error('No data');
      return res.data as T;
    },
    []
  );

  const deleteQuery = useCallback<(args: DeleteQueryArgs) => Promise<string[]>>(
    async ({ id }) => {
      const data = await executeMutation<RemoveInsightsQueryMutation>(() => runRemove({ id }));
      const ids = data.removeInsightsQuery?.ids;
      if (!ids) throw new Error('No data');
      return ids;
    },
    [executeMutation, runRemove]
  );

  const saveQuery = useCallback<(args: SaveQueryArgs) => Promise<InsightsQuery>>(
    async ({ name, query }) => {
      const data = await executeMutation<CreateInsightsQueryMutation>(() =>
        runCreate({ input: { name, sql: query } })
      );
      return data.createInsightsQuery;
    },
    [executeMutation, runCreate]
  );

  const updateQuery = useCallback<(args: UpdateQueryArgs) => Promise<InsightsQuery>>(
    async ({ id, name, query }) => {
      const data = await executeMutation<UpdateInsightsQueryMutation>(() =>
        runUpdate({ id, input: { name, sql: query } })
      );
      return data.updateInsightsQuery;
    },
    [executeMutation, runUpdate]
  );

  return { deleteQuery, saveQuery, updateQuery };
}
