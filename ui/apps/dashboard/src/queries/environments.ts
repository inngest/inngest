import { useQuery, type UseQueryResponse } from 'urql';

import { graphql } from '@/gql';
import { workspacesToEnvironments, type Environment } from '@/utils/environments';

type UseEnvironmentParams = {
  environmentSlug: string;
};

export const useEnvironment = ({
  environmentSlug,
}: UseEnvironmentParams): UseQueryResponse<Environment> => {
  const [{ data: environments, fetching, error, stale }, refetch] = useEnvironments();

  const environment = environments?.find((e) => {
    return e.slug === environmentSlug;
  });

  return [{ data: environment, fetching, error, stale }, refetch];
};

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
      functionCount
      isAutoArchiveEnabled
      lastDeployedAt
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
