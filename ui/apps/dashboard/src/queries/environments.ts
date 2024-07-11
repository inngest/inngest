import { useQuery, type UseQueryResponse } from 'urql';

import { graphql } from '@/gql';
import type { Workspace } from '@/gql/graphql';
import { workspacesToEnvironments, type Environment } from '@/utils/environments';

export function useProductionEnvironment(): UseQueryResponse<Environment> {
  const [{ fetching, data, error, stale }, refetch] = useQuery({
    query: GetProductionWorkspaceDocument,
    requestPolicy: 'cache-first',
  });

  const environment = workspacesToEnvironments(
    (data?.productionWorkspace ? [data?.productionWorkspace] : []) as Workspace[]
  )[0];

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

  const environments = workspacesToEnvironments(data?.workspaces ?? []);

  return [{ data: environments, fetching, error, stale }, refetch];
};

export const useEnvironment = (environmentSlug: string): UseQueryResponse<Environment> => {
  const [{ fetching, data, error, stale }, refetch] = useQuery({
    query: GetEnvironmentBySlugDocument,
    requestPolicy: 'cache-first',
    variables: { slug: environmentSlug },
  });

  const environment = workspacesToEnvironments(
    (data?.workspaceBySlug ? [data.workspaceBySlug] : []) as Workspace[]
  )[0];

  return [{ data: environment, fetching, error, stale }, refetch];
};
