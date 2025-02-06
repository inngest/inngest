import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query LatestUnattachedSync($envID: ID!) {
    environment: workspace(id: $envID) {
      unattachedSyncs(first: 1) {
        lastSyncedAt
      }
    }
  }
`);

export function useLatestUnattachedSync({ envID }: { envID: string }) {
  const res = useGraphQLQuery({
    pollIntervalInMilliseconds: 10_000,
    query,
    variables: { envID },
  });

  // We are flattening the latestSync data to match the structure used in the DevServer
  if (res.data) {
    let latestUnattachedSyncTime;
    if (res.data.environment.unattachedSyncs[0]) {
      latestUnattachedSyncTime = new Date(res.data.environment.unattachedSyncs[0].lastSyncedAt);
    }

    return {
      ...res,
      data: latestUnattachedSyncTime,
    };
  }

  return {
    ...res,
    data: undefined,
  };
}
