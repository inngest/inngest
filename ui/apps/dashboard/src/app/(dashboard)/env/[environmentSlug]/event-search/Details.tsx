import { useState } from 'react';
import { EventDetails } from '@inngest/components/EventDetails';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import { SlideOver } from './SlideOver';

const eventQuery = graphql(`
  query GetEventSearchEvent($envID: ID!, $eventID: ULID!) {
    environment: workspace(id: $envID) {
      event: archivedEvent(id: $eventID) {
        id
        name
        payload: event
        receivedAt
        runs: functionRuns {
          id
          output
          status
        }
      }
    }
  }
`);

type Props = {
  envID: string;
  eventID: string | undefined;
  onClose: () => void;
};

export function Details({ envID, eventID, onClose }: Props) {
  const [selectedRunID, setSelectedRunID] = useState<string | undefined>(undefined);

  const isOpen = Boolean(eventID);

  const [{ data, error, fetching }] = useQuery({
    query: eventQuery,
    variables: {
      envID,
      eventID: eventID || 'unset',
    },
    pause: !isOpen,
  });

  if (error) {
    throw error;
  }

  let event: React.ComponentProps<typeof EventDetails>['event'] | undefined;
  let runs: React.ComponentProps<typeof EventDetails>['functionRuns'] | undefined;
  if (isOpen && !fetching) {
    if (!data?.environment.event) {
      // Should be unreachable.
      throw new Error('missing data');
    }

    event = {
      ...data.environment.event,
      receivedAt: new Date(data.environment.event.receivedAt),
    };
    runs = data.environment.event.runs.map((run) => {
      return {
        ...run,
        name: 'Foo',
        status: 'RUNNING',
      } as const;
    });
  }

  return (
    <SlideOver isOpen={isOpen} onClose={onClose} size={selectedRunID ? 'large' : 'small'}>
      {event && runs && (
        <EventDetails
          event={event}
          functionRuns={runs}
          onFunctionRunClick={setSelectedRunID}
          selectedRunID={selectedRunID}
        />
      )}
    </SlideOver>
  );
}
