import { useCallback } from 'react';
import type {
  GetRunPayload,
  GetRunResult,
} from '@inngest/components/SharedContext/useGetRun';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { getFragmentData, graphql } from '@/gql';
import { indexTicks, mergeTicks } from './traceTicks';

const traceDetailsFragment = graphql(`
  fragment TraceDetails on RunTraceSpan {
    name
    status
    attempts
    queuedAt
    scheduledAt
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
    metadata {
      kind
      scope
      values
      updatedAt
    }
    outputID
    stepID
    spanID
    groupID
    stepOp
    stepType
    groupKind
    origin
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
    response {
      statusCode
      headers
    }
  }
`);

// Cloud caps query complexity at 300 and each level costs ~58, so this stops
// at 5 levels. Deeper spans (extra machines, command retries, failed attempts
// in CI traces) aren't shown until span children are loaded lazily.
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
      }
    }
  }
`);

// Ticks (the dots for messages arriving at an inbox) can't join the query
// above: it already costs 294 of the 300 Cloud allows, and ticks on five levels
// cost 30 more. They come from this lighter query instead, to the same depth,
// and mergeTicks joins them to the trace by span ID. The budget is checked in
// src/gql/runTraceComplexity.test.ts.
const ticksQuery = graphql(`
  query GetRunTraceTicks($envID: ID!, $runID: String!, $preview: Boolean) {
    workspace(id: $envID) {
      run(runID: $runID) {
        trace(preview: $preview) {
          spanID
          ticks {
            id
            sources
            at
            readAt
          }
          childrenSpans {
            spanID
            ticks {
              id
              sources
              at
              readAt
            }
            childrenSpans {
              spanID
              ticks {
                id
                sources
                at
                readAt
              }
              childrenSpans {
                spanID
                ticks {
                  id
                  sources
                  at
                  readAt
                }
                childrenSpans {
                  spanID
                  ticks {
                    id
                    sources
                    at
                    readAt
                  }
                }
              }
            }
          }
        }
      }
    }
  }
`);

export function useGetRun() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async ({ runID, preview }: GetRunPayload): Promise<GetRunResult> => {
      const variables = { envID, runID, preview: preview ?? false };
      const context = { requestPolicy: 'network-only' } as const;

      const [result, ticksResult] = await Promise.all([
        client.query(query, variables, context).toPromise(),
        client.query(ticksQuery, variables, context).toPromise(),
      ]);

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

      // Ticks only draw dots, so a failed ticks query leaves the dots out
      // instead of hiding the run.
      if (ticksResult.error) {
        console.warn(
          'Could not load the ticks of the run trace',
          ticksResult.error,
        );
      }

      const ticks = indexTicks(ticksResult.data?.workspace.run?.trace);

      return {
        loading: false,
        error: undefined,
        data: {
          ...run,
          app: fn.app,
          id: runID,
          fn,
          trace: mergeTicks(trace, ticks),
        },
      };
    },
    [envID, client],
  );
}
