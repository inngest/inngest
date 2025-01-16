import { useCallback } from 'react';
import type { Result } from '@inngest/components/types/functionRun';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

const query = graphql(`
  query TraceResult($envID: ID!, $traceID: String!) {
    workspace(id: $envID) {
      runTraceSpanOutputByID(outputID: $traceID) {
        data
        input
        error {
          message
          name
          stack
        }
      }
    }
  }
`);

export function useGetTraceResult(): (traceID: string) => Promise<Result> {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async (traceID: string) => {
      let res;
      try {
        res = await client
          .query(query, { envID: envID, traceID }, { requestPolicy: 'network-only' })
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

      return {
        ...res.data.workspace.runTraceSpanOutputByID,
      };
    },
    [client, envID]
  );
}
