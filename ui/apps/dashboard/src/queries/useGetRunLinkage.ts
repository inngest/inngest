import { useCallback } from 'react';
import type {
  GetRunLinkagePayload,
  GetRunLinkageResult,
} from '@inngest/components/SharedContext/useGetRunLinkage';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

const query = graphql(`
  query GetRunLinkage($envID: ID!, $runID: String!) {
    workspace(id: $envID) {
      run(runID: $runID) {
        defers {
          deferID
          function {
            name
            slug
          }
          hashedDeferID
          run {
            id
            status
          }
          status
        }
        siblingDefers {
          deferID
          function {
            name
            slug
          }
          hashedDeferID
          run {
            id
            status
          }
          status
        }
        deferredFrom {
          function {
            name
            slug
          }
          run {
            id
            status
          }
        }
      }
    }
  }
`);

export function useGetRunLinkage() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async ({ runID }: GetRunLinkagePayload): Promise<GetRunLinkageResult> => {
      const result = await client
        .query(query, { envID, runID }, { requestPolicy: 'network-only' })
        .toPromise();

      if (result.error) {
        return { loading: false, error: result.error };
      }
      const run = result.data?.workspace.run;
      if (!run) {
        return { loading: false, error: new Error('missing run') };
      }
      return {
        loading: false,
        data: {
          defers: run.defers,
          siblingDefers: run.siblingDefers,
          deferredFrom: run.deferredFrom,
        },
      };
    },
    [envID, client],
  );
}
