import { useQuery } from "@tanstack/react-query";
import { useClient } from "urql";

import { graphql } from "@/gql";

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
  const client = useClient();

  return useQuery({
    queryKey: ["latestUnattachedSync", envID],
    queryFn: async () => {
      const result = await client.query(query, { envID }).toPromise();

      if (result.error) {
        throw result.error;
      }

      if (!result.data?.environment.unattachedSyncs[0]) {
        return null;
      }

      return new Date(result.data.environment.unattachedSyncs[0].lastSyncedAt);
    },
    refetchInterval: 10000,
  });
}
