'use client';

import { useCallback } from 'react';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import type {
  CreateInsightsQueryMutation,
  RemoveInsightsQueryMutation,
  UpdateInsightsQueryMutation,
} from '@/gql/graphql';
import { toLocalQuery } from '../queries';
import type { Query as LocalQuery } from '../types';

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
  saveQuery: (args: SaveQueryArgs) => Promise<LocalQuery>;
  updateQuery: (args: UpdateQueryArgs) => Promise<LocalQuery>;
};

export function useModifySavedQueries(): UseModifySavedQueriesReturn {
  const [, runCreate] = useMutation(createInsightsQueryDocument);
  const [, runRemove] = useMutation(removeInsightsQueryDocument);
  const [, runUpdate] = useMutation(updateInsightsQueryDocument);

  const exec = useCallback(
    async <T>(fn: () => Promise<{ data?: T; error?: unknown }>): Promise<T> => {
      const res = await fn();
      if ((res as any).error) throw (res as any).error;
      if (!res.data) throw new Error('No data');
      return res.data as T;
    },
    []
  );

  const saveQuery = useCallback<(args: SaveQueryArgs) => Promise<LocalQuery>>(
    async ({ name, query }) => {
      const data = await exec<CreateInsightsQueryMutation>(() =>
        runCreate({ input: { name, sql: query } })
      );
      return toLocalQuery(data.createInsightsQuery);
    },
    [exec, runCreate]
  );

  const updateQuery = useCallback<(args: UpdateQueryArgs) => Promise<LocalQuery>>(
    async ({ id, name, query }) => {
      const data = await exec<UpdateInsightsQueryMutation>(() =>
        runUpdate({ id, input: { name, sql: query } })
      );
      return toLocalQuery(data.updateInsightsQuery);
    },
    [exec, runUpdate]
  );

  const deleteQuery = useCallback<(args: DeleteQueryArgs) => Promise<string[]>>(
    async ({ id }) => {
      const data = await exec<RemoveInsightsQueryMutation>(() => runRemove({ id }));
      const ids = data.removeInsightsQuery?.ids;
      if (!ids) throw new Error('No data');
      return ids;
    },
    [exec, runRemove]
  );

  return { saveQuery, updateQuery, deleteQuery };
}
