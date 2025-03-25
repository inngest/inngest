import { useMemo } from 'react';
import { baseInitialFetchFailed, type FetchResult } from '@inngest/components/types/fetch';
import type { Function } from '@inngest/components/types/function';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import type { FunctionVersion } from '@inngest/components/types/functionVersion';

import { graphql } from '@/gql';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';

const runQuery = graphql(`
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

  const res = useSkippableGraphQLQuery({
    query: runQuery,
    skip,
    variables: {
      envID,
      functionID: functionID ?? 'unset',
      runID: runID ?? 'unset',
    },
  });

  // Transform the API data into the shape our shared UI expects.
  const data = useMemo((): Data | Error => {
    const func = res.data?.environment.function ?? undefined;
    const run = res.data?.environment.function?.run ?? undefined;

    if (!func) {
      return new Error('result is missing function data');
    }
    if (!run) {
      return new Error('result is missing run data');
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

  if (!res.data) {
    return {
      ...res,
      data: undefined,
    };
  }

  if (data instanceof Error) {
    // Should be unreachable
    return {
      ...baseInitialFetchFailed,
      error: data,
      refetch: res.refetch,
    };
  }

  return {
    ...res,
    data,
  };
}
