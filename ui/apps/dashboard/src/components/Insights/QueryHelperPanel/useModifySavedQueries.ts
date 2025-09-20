'use client';

import { useCallback } from 'react';
import { CombinedError, useMutation } from 'urql';

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
  mutation RemoveInsightsQuery($id: ULID!) {
    removeInsightsQuery(id: $id) {
      ids
    }
  }
`);

const updateInsightsQueryDocument = graphql(`
  mutation UpdateInsightsQuery($id: ULID!, $input: UpdateInsightsQuery!) {
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

export type MutationResult<T> = { ok: true; data: T } | { ok: false; error: 'unique' | 'other' };

type UseModifySavedQueriesReturn = {
  deleteQuery: (args: DeleteQueryArgs) => Promise<MutationResult<string[]>>;
  saveQuery: (args: SaveQueryArgs) => Promise<MutationResult<InsightsQuery>>;
  updateQuery: (args: UpdateQueryArgs) => Promise<MutationResult<InsightsQuery>>;
};

export function useModifySavedQueries(): UseModifySavedQueriesReturn {
  const [, runCreate] = useMutation(createInsightsQueryDocument);
  const [, runRemove] = useMutation(removeInsightsQueryDocument);
  const [, runUpdate] = useMutation(updateInsightsQueryDocument);

  const executeMutation = useCallback(
    async <T>(
      fn: () => Promise<{ data?: T; error?: CombinedError }>
    ): Promise<{ ok: true; data: T } | { ok: false; error: CombinedError }> => {
      const res = await fn();
      if (res.error) return { ok: false, error: res.error };
      if (res.data === undefined) return { ok: false, error: new CombinedError({}) };
      return { ok: true, data: res.data };
    },
    []
  );

  const deleteQuery = useCallback<(args: DeleteQueryArgs) => Promise<MutationResult<string[]>>>(
    async ({ id }) => {
      const result = await executeMutation<RemoveInsightsQueryMutation>(() => runRemove({ id }));
      if (!result.ok) return { ok: false, error: 'other' };

      const ids = result.data.removeInsightsQuery?.ids;
      if (ids === undefined) return { ok: false, error: 'other' };

      return { ok: true, data: ids };
    },
    [executeMutation, runRemove]
  );

  const saveQuery = useCallback<(args: SaveQueryArgs) => Promise<MutationResult<InsightsQuery>>>(
    async ({ name, query }) => {
      const result = await executeMutation<CreateInsightsQueryMutation>(() =>
        runCreate({ input: { name, sql: query } })
      );
      if (!result.ok) return { ok: false, error: mapErrorToTag(result.error) };

      return { ok: true, data: result.data.createInsightsQuery };
    },
    [executeMutation, runCreate]
  );

  const updateQuery = useCallback<
    (args: UpdateQueryArgs) => Promise<MutationResult<InsightsQuery>>
  >(
    async ({ id, name, query }) => {
      const result = await executeMutation<UpdateInsightsQueryMutation>(() =>
        runUpdate({ id, input: { name, sql: query } })
      );
      if (!result.ok) return { ok: false, error: mapErrorToTag(result.error) };

      return { ok: true, data: result.data.updateInsightsQuery };
    },
    [executeMutation, runUpdate]
  );

  return { deleteQuery, saveQuery, updateQuery };
}

function mapErrorToTag(error: CombinedError): 'unique' | 'other' {
  const isUnique = error.graphQLErrors.some((g) =>
    g.message.includes('uniq_insights_queries_name')
  );
  return isUnique ? 'unique' : 'other';
}
