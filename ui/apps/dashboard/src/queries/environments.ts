import { useMemo } from 'react';
import { useQuery, type UseQueryResponse } from 'urql';

import { graphql } from '@/gql';
import type { Workspace } from '@/gql/graphql';
import {
  workspaceToEnvironment,
  workspacesToEnvironments,
  type Environment,
} from '@/utils/environments';

export function useDefaultEnvironment(): UseQueryResponse<Environment> {
  const [{ fetching, data, error, stale }, refetch] = useQuery({
    query: GetDefaultEnvironmentDocument,
    requestPolicy: 'cache-first',
  });

  const environment = useMemo(() => {
    if (!data?.defaultEnv) {
      return;
    }

    return workspaceToEnvironment(data.defaultEnv as Workspace);
  }, [data?.defaultEnv]);

  return [{ data: environment, fetching, error, stale }, refetch];
}

const GetEnvironmentsDocument = graphql(`
  query GetEnvironments {
    workspaces {
      id
      name
      slug
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
    envBySlug(slug: $slug) {
      id
      name
      slug
      parentID
      test
      type
      createdAt
      lastDeployedAt
      isArchived
      isAutoArchiveEnabled
      webhookSigningKey
    }
  }
`);

const GetDefaultEnvironmentDocument = graphql(`
  query GetDefaultEnvironment {
    defaultEnv {
      id
      name
      slug
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
    if (!data?.envBySlug) {
      return;
    }

    return workspaceToEnvironment(data.envBySlug as Workspace);
  }, [data?.envBySlug]);

  return [{ data: environment, fetching, error, stale }, refetch];
};
