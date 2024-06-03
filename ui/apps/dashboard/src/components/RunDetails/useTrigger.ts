import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query GetRunTraceTrigger($envID: ID!, $runID: String!) {
    workspace(id: $envID) {
      runTrigger(runID: $runID) {
        IDs
        payloads
        timestamp
        eventName
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

  return {
    ...res,
    data: {
      trigger: runTrigger,
    },
  };
}
