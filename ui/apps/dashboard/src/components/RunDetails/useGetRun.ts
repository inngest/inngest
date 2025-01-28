import { useCallback } from 'react';
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

  return useCallback(
    async (runID: string) => {
      let res;
      try {
        res = await client
          .query(query, { envID, runID }, { requestPolicy: 'network-only' })
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
        ...run,
        app: fn.app,
        id: runID,
        fn,
        trace,
      };
    },
    [client, envID]
  );
}
