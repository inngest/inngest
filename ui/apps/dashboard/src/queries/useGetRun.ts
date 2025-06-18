import { useState } from 'react';
import type { GetRunPayload } from '@inngest/components/SharedContext/useGetRun';
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
  query GetRunTrace($envID: ID!, $runID: String!) {
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
        trace {
          ...TraceDetails
          childrenSpans {
            ...TraceDetails
            childrenSpans {
              ...TraceDetails
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
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error>();

  return async ({ runID }: GetRunPayload) => {
    setLoading(true);
    setError(undefined);
    const res = await client
      .query(query, { envID, runID }, { requestPolicy: 'network-only' })
      .toPromise();

    if (res.error) {
      throw res.error;
    }
    if (!res.data) {
      throw new Error('no data returned');
    }

    const { run } = res.data.workspace;
    if (!run) {
      throw new Error('missing run');
    }

    const fn = run.function;

    const trace = getFragmentData(traceDetailsFragment, run.trace);
    if (!trace) {
      throw new Error('missing trace');
    }

    return {
      loading,
      error,
      data: {
        ...run,
        app: fn.app,
        id: runID,
        fn,
        trace,
      },
    };
  };
}
