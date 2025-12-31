import { useCallback } from 'react';
import type {
  GetTraceResultPayload,
  TraceResult,
} from '@inngest/components/SharedContext/useGetTraceResult';
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
          cause
        }
      }
    }
  }
`);

export function useGetTraceResult(): (
  payload: GetTraceResultPayload,
) => Promise<TraceResult> {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async ({ traceID }: GetTraceResultPayload) => {
      let res;
      try {
        res = await client
          .query(
            query,
            { envID: envID, traceID },
            { requestPolicy: 'network-only' },
          )
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
    [client, envID],
  );
}
