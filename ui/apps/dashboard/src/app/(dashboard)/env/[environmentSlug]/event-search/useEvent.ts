import { useMemo } from 'react';
import type { Event } from '@inngest/components/types/event';
import {
  baseFetchFailed,
  baseFetchLoading,
  baseFetchSkipped,
  baseFetchSucceeded,
  type FetchResult,
} from '@inngest/components/types/fetch';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import { useQuery } from 'urql';

import { graphql } from '@/gql';

const eventQuery = graphql(`
  query GetEventSearchEvent($envID: ID!, $eventID: ULID!) {
    environment: workspace(id: $envID) {
      event: archivedEvent(id: $eventID) {
        id
        name
        payload: event
        receivedAt
        runs: functionRuns {
          function {
            name
          }
          id
          output
          status
        }
      }
    }
  }
`);

type Data = {
  event: Event;
  runs: Pick<FunctionRun, 'id' | 'name' | 'output' | 'status'>[];
};

export function useEvent({
  envID,
  eventID,
}: {
  envID: string;
  eventID: string | undefined;
}): FetchResult<Data, { skippable: true }> {
  const skip = !eventID;

  const [res] = useQuery({
    query: eventQuery,
    variables: {
      envID,
      eventID: eventID ?? 'unset',
    },
    pause: !eventID,
  });

  // In addition to memoizing, this hook will also transform the API data into
  // the shape our shared UI expects.
  const data = useMemo((): Data | undefined => {
    const event = res.data?.environment.event ?? undefined;

    if (!event) {
      return undefined;
    }

    const runs: Data['runs'] = (event.runs ?? []).map((run) => {
      return {
        id: run.id,
        name: run.function.name,
        output: run.output,
        status: run.status,
      };
    });

    return {
      event: {
        ...event,
        receivedAt: new Date(event.receivedAt),
      },
      runs,
    };
  }, [res.data?.environment.event]);

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
