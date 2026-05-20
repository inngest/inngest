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
          id
          userDeferID
          fnSlug
          status
          run {
            id
            status
            function {
              name
              slug
            }
          }
        }
        deferredFrom {
          parentRunID
          parentRun {
            id
            status
            function {
              name
              slug
            }
            defers {
              id
              userDeferID
              fnSlug
              status
              run {
                id
                status
                function {
                  name
                  slug
                }
              }
            }
          }
        }
        invokedFrom {
          parentRunID
          stepName
          parentRun {
            id
            status
            function {
              name
              slug
            }
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
        return {
          loading: false,
          error: result.error,
          data: undefined,
        };
      }

      if (!result.data) {
        return {
          loading: false,
          error: new Error('no data returned'),
          data: undefined,
        };
      }

      const { run } = result.data.workspace;
      if (!run) {
        return {
          loading: false,
          error: new Error('missing run'),
          data: undefined,
        };
      }

      return {
        loading: false,
        error: undefined,
        data: {
          defers: run.defers,
          deferredFrom: run.deferredFrom,
          invokedFrom: run.invokedFrom,
        },
      };
    },
    [envID, client],
  );
}
