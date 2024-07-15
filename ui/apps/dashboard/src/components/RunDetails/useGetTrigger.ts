import { useCallback } from 'react';
import type { Trigger } from '@inngest/components/TriggerDetails/TriggerDetails';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

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

export function useGetTrigger(): (runID: string) => Promise<Trigger> {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async (runID: string) => {
      let res;
      try {
        res = await client
          .query(query, { envID: envID, runID }, { requestPolicy: 'network-only' })
          .toPromise();
      } catch (e) {
        if (e instanceof Error) {
          throw e;
        }
        throw new Error('unknown error');
      }
      if (res.error) {
        throw res.error;
      }
      if (!res.data) {
        throw new Error('no data returned');
      }
      return res.data.workspace.runTrigger;
    },
    [client, envID]
  );
}
