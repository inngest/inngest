import { useCallback } from 'react';
import type { GetRunPayload, GetRunResult } from '@inngest/components/SharedContext/useGetRun';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { getFragmentData, graphql } from '@/gql';

const traceDetailsFragment = graphql(`
  fragment TraceDetails on RunTraceSpan {
    name
    status
    attempts
    queuedAt
    startedAt
    endedAt
    isRoot
    isUserland
    userlandSpan {
      spanName
      spanKind
      serviceName
      scopeName
      scopeVersion
      spanAttrs
      resourceAttrs
    }
    outputID
    stepID
    spanID
    stepOp
    stepType
    stepInfo {
      __typename
      ... on InvokeStepInfo {
        triggeringEventID
        functionID
        timeout
        returnEventID
        runID
        timedOut
      }
      ... on SleepStepInfo {
        sleepUntil
      }
      ... on WaitForEventStepInfo {
        eventName
        expression
        timeout
        foundEventID
        timedOut
      }
    }
  }
`);

const query = graphql(`
  query GetRunTrace($envID: ID!, $runID: String!, $preview: Boolean) {
    workspace(id: $envID) {
      run(runID: $runID) {
        function {
          app {
            name
            externalID
          }
          id
          name
          slug
        }
        status
        trace(preview: $preview) {
          ...TraceDetails
          childrenSpans {
            ...TraceDetails
            childrenSpans {
              ...TraceDetails
              childrenSpans {
                ...TraceDetails
                childrenSpans {
                  ...TraceDetails
                }
              }
            }
          }
        }
        hasAI
      }
    }
  }
`);

export function useGetRun() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async ({ runID, preview }: GetRunPayload): Promise<GetRunResult> => {
      const result = await client
        .query(
          query,
          { envID, runID, preview: preview ?? false },
          { requestPolicy: 'network-only' }
        )
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

      const fn = run.function;
      const trace = getFragmentData(traceDetailsFragment, run.trace);

      if (!trace) {
        return {
          loading: false,
          error: new Error('missing trace'),
          data: undefined,
        };
      }

      return {
        loading: false,
        error: undefined,
        data: {
          ...run,
          app: fn.app,
          id: runID,
          fn,
          trace,
        },
      };
    },
    [envID, client]
  );
}
