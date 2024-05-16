import { baseInitialFetchFailed } from '@inngest/components/types/fetch';

import { getFragmentData, graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

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
          }
          name
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
      }
    }
  }
`);

export function useRun({ envID, runID }: { envID: string; runID: string }) {
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

  const { run } = res.data.workspace;
  if (!run) {
    return {
      ...baseInitialFetchFailed,
      error: new Error('missing run'),
    };
  }

  if (!run.trace) {
    return {
      ...baseInitialFetchFailed,
      error: new Error('missing trace'),
    };
  }

  if (!run.function) {
    return {
      ...baseInitialFetchFailed,
      error: new Error('missing function'),
    };
  }

  return {
    ...res,
    data: {
      run: {
        ...run,
        function: run.function,
      },
      trace: getFragmentData(traceDetailsFragment, run.trace),
    },
  };
}
