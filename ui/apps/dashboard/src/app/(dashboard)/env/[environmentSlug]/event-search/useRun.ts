import { useMemo } from 'react';
import type { Event } from '@inngest/components/types/event';
import {
  baseFetchFailed,
  baseFetchLoading,
  baseFetchSkipped,
  baseFetchSucceeded,
  type FetchResult,
} from '@inngest/components/types/fetch';
import type { Function } from '@inngest/components/types/function';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import type { FunctionVersion } from '@inngest/components/types/functionVersion';
import { useQuery } from 'urql';

import { graphql } from '@/gql';

const eventQuery = graphql(`
  query GetEventSearchRun($envID: ID!, $functionID: ID!, $runID: ULID!) {
    environment: workspace(id: $envID) {
      function: workflow(id: $functionID) {
        name
        run(id: $runID) {
          canRerun
          history {
            attempt
            cancel {
              eventID
              expression
              userID
            }
            createdAt
            functionVersion
            groupID
            id
            sleep {
              until
            }
            stepName
            type
            url
            waitForEvent {
              eventName
              expression
              timeout
            }
            waitResult {
              eventID
              timeout
            }
          }
          id
          status
          startedAt
          endedAt
          output
          version: workflowVersion {
            deploy {
              id
              createdAt
            }
            triggers {
              eventName
              schedule
            }
            url
            validFrom
            version
          }
        }
      }
    }
  }
`);

type Data = {
  func: Pick<Function, 'name' | 'triggers' | 'version'>;
  functionVersion: Pick<FunctionVersion, 'url' | 'version'> | undefined;
  run: Pick<FunctionRun, 'endedAt' | 'history' | 'id' | 'output' | 'startedAt' | 'status'>;
};

export function useRun({
  envID,
  functionID,
  runID,
}: {
  envID: string;
  functionID: string | undefined;
  runID: string | undefined;
}): FetchResult<Data, { skippable: true }> {
  const skip = !functionID || !runID;

  const [res] = useQuery({
    query: eventQuery,
    variables: {
      envID,
      functionID: functionID ?? 'unset',
      runID: runID ?? 'unset',
    },
    pause: skip,
  });

  // In addition to memoizing, this hook will also transform the API data into
  // the shape our shared UI expects.
  const data = useMemo((): Data | undefined => {
    const func = res.data?.environment.function ?? undefined;
    const run = res.data?.environment.function?.run ?? undefined;

    if (!func || !run) {
      return undefined;
    }

    const triggers = (run.version?.triggers ?? []).map((trigger) => {
      return {
        type: trigger.schedule ? 'CRON' : 'EVENT',
        value: trigger.schedule ?? trigger.eventName ?? '',
      } as const;
    });

    return {
      func: {
        ...func,
        triggers,
        version: run.version?.version ?? null,
      },
      functionVersion: run.version ?? undefined,
      run: {
        ...run,
        endedAt: run.endedAt ? new Date(run.endedAt) : null,
        startedAt: run.startedAt ? new Date(run.startedAt) : null,
      },
    };
  }, [res.data?.environment.function]);

  if (res.fetching) {
    return baseFetchLoading;
  }

  if (skip) {
    return baseFetchSkipped;
  }

  if (res.error) {
    return {
      ...baseFetchFailed,
      error: new Error(res.error.message),
    };
  }

  if (!data) {
    // Should be unreachable.
    return {
      ...baseFetchFailed,
      error: new Error('finished loading but missing data'),
    };
  }

  return {
    ...baseFetchSucceeded,
    data,
  };
}
