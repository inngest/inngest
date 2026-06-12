import { useCallback } from 'react';
import type {
  GetRunLinkagePayload,
  GetRunLinkageResult,
  RunDeferSummary,
  RunDeferredFromSummary,
} from '@inngest/components/SharedContext/useGetRunLinkage';
import { gql, type TypedDocumentNode, useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';

type GetRunLinkageQueryVariables = {
  envID: string;
  runID: string;
};

type GetRunLinkageQuery = {
  workspace: {
    run: {
      defers: RunDeferSummary[];
      siblingDefers: RunDeferSummary[];
      deferredFrom: RunDeferredFromSummary[];
    } | null;
  };
};

const query: TypedDocumentNode<
  GetRunLinkageQuery,
  GetRunLinkageQueryVariables
> = gql`
  query GetRunLinkage($envID: ID!, $runID: String!) {
    workspace(id: $envID) {
      run(runID: $runID) {
        defers {
          ...RunDeferSummaryFields
        }
        siblingDefers {
          ...RunDeferSummaryFields
        }
        deferredFrom {
          runID
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

  fragment RunDeferSummaryFields on RunDefer {
    hashedDeferID
    userlandDeferID
    fnSlug
    status
    function {
      name
      slug
    }
    run {
      id
      status
    }
  }
`;

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

      const run = result.data?.workspace.run;
      if (!run) {
        return {
          loading: false,
          error: new Error('missing run'),
          data: undefined,
        };
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
    [client, envID],
  );
}
