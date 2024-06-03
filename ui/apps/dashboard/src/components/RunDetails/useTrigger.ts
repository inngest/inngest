import { baseInitialFetchFailed } from '@inngest/components/types/fetch';

import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query GetRunTraceTrigger($envID: ID!, $runID: String!) {
    workspace(id: $envID) {
      runTrigger(runID: $runID) {
        IDs
        payloads
        timestamp
        isBatch
        batchID
        cron
      }
    }
  }
`);

export function useTrigger({ envID, runID }: { envID: string; runID: string }) {
  const res = useGraphQLQuery({
    query,
    variables: {
      envID,
      runID,
    },
  });

  if (!res.data) {
    return res;
  }

  const { runTrigger } = res.data.workspace;
  if (!runTrigger) {
    return {
      ...baseInitialFetchFailed,
      error: new Error('missing run trigger'),
    };
  }

  return {
    ...res,
    data: {
      trigger: runTrigger,
    },
  };
}
