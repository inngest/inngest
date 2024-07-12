import { useMemo } from 'react';
import { useQuery, type UseQueryResponse } from 'urql';

import { graphql } from '@/gql';
import type { Workspace } from '@/gql/graphql';
import {
  workspaceToEnvironment,
  workspacesToEnvironments,
  type Environment,
} from '@/utils/environments';

export function useProductionEnvironment(): UseQueryResponse<Environment> {
  const [{ fetching, data, error, stale }, refetch] = useQuery({
    query: GetProductionWorkspaceDocument,
    requestPolicy: 'cache-first',
  });

  const environment = useMemo(() => {
    if (!data?.productionWorkspace) {
      return;
    }

    return workspaceToEnvironment(data.productionWorkspace as Workspace);
  }, [data?.productionWorkspace]);

  return [{ data: environment, fetching, error, stale }, refetch];
}

const GetEnvironmentsDocument = graphql(`
  query GetEnvironments {
    workspaces {
      id
      name
      parentID
      test
      type
      webhookSigningKey
      createdAt
      isArchived
      isAutoArchiveEnabled
      lastDeployedAt
    }
  }
`);

const GetEnvironmentBySlugDocument = graphql(`
  query GetEnvironmentBySlug($slug: String!) {
    workspaceBySlug(slug: $slug) {
      id
      name
      parentID
      test
      type
      createdAt
      lastDeployedAt
      isArchived
      isAutoArchiveEnabled
    }
  }
`);

const GetProductionWorkspaceDocument = graphql(`
  query GetProductionWorkspace {
    productionWorkspace {
      id
      name
      parentID
      test
      type
      createdAt
      lastDeployedAt
      isArchived
      isAutoArchiveEnabled
    }
  }
`);

export const useEnvironments = (): UseQueryResponse<Environment[]> => {
  const [{ fetching, data, error, stale }, refetch] = useQuery({
    query: GetEnvironmentsDocument,
    requestPolicy: 'cache-first',
  });

  const environments = useMemo(() => {
    return workspacesToEnvironments(data?.workspaces ?? []);
  }, [data?.workspaces]);

  return [{ data: environments, fetching, error, stale }, refetch];
};

export const useEnvironment = (environmentSlug: string): UseQueryResponse<Environment> => {
  const [{ fetching, data, error, stale }, refetch] = useQuery({
    query: GetEnvironmentBySlugDocument,
    requestPolicy: 'cache-first',
    variables: { slug: environmentSlug },
  });

  const environment = useMemo(() => {
    if (!data?.workspaceBySlug) {
      return;
    }

    return workspaceToEnvironment(data.workspaceBySlug as Workspace);
  }, [data?.workspaceBySlug]);

  return [{ data: environment, fetching, error, stale }, refetch];
};
