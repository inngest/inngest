import { useMemo } from 'react';
import type { Event } from '@inngest/components/types/event';
import { baseInitialFetchFailed } from '@inngest/components/types/fetch';
import type { FunctionRun } from '@inngest/components/types/functionRun';

import { graphql } from '@/gql';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';

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
            id
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
  runs: Pick<FunctionRun, 'functionID' | 'id' | 'name' | 'output' | 'status'>[];
};

export function useEvent({ envID, eventID }: { envID: string; eventID: string | undefined }) {
  const skip = !eventID;

  const res = useSkippableGraphQLQuery({
    query: eventQuery,
    skip,
    variables: {
      envID,
      eventID: eventID ?? 'unset',
    },
  });

  // Transform the API data into the shape our shared UI expects.
  const data = useMemo((): Data | Error => {
    const event = res.data?.environment.event ?? undefined;

    if (!event) {
      return new Error('result is missing event data');
    }

    const runs: Data['runs'] = event.runs.map((run) => {
      return {
        functionID: run.function.id,
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
    };
  }

  return {
    ...res,
    data,
  };
}
