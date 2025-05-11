import { useCallback } from 'react';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

export const eventsQuery = graphql(`
  query GetEventsV2(
    $envID: ID!
    $cursor: String
    $startTime: Time!
    $endTime: Time
    $celQuery: String = null
    $eventNames: [String!] = null
  ) {
    environment: workspace(id: $envID) {
      eventsV2(
        first: 30
        after: $cursor
        filter: { from: $startTime, until: $endTime, query: $celQuery, eventNames: $eventNames }
      ) {
        edges {
          node {
            name
            id
            receivedAt
            runs {
              status
              id
              startedAt
              endedAt
              function {
                name
                slug
              }
            }
          }
        }
        totalCount
        pageInfo {
          hasNextPage
          endCursor
          hasPreviousPage
          startCursor
        }
      }
    }
  }
`);

type EventsQueryVariables = {
  eventNames: string[] | null;
  cursor: string | null;
  source?: string;
  startTime: string;
  endTime: string | null;
  celQuery?: string;
};

export function useEvents() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async ({ cursor, endTime, source, eventNames, startTime, celQuery }: EventsQueryVariables) => {
      console.log(source);
      const result = await client
        .query(
          eventsQuery,
          {
            envID,
            startTime,
            endTime,
            cursor,
            celQuery,
            eventNames,
          },
          { requestPolicy: 'network-only' }
        )
        .toPromise();

      if (result.error) {
        throw new Error(result.error.message);
      }

      if (!result.data) {
        throw new Error('no data returned');
      }

      const eventsData = result.data.environment.eventsV2;
      const events = eventsData.edges.map(({ node }) => ({
        ...node,
        receivedAt: new Date(node.receivedAt),
        runs: node.runs.map((run) => ({
          fnName: run.function.name,
          fnSlug: run.function.slug,
          status: run.status,
          id: run.id,
          completedAt: run.endedAt ? new Date(run.endedAt) : undefined,
          startedAt: run.startedAt ? new Date(run.startedAt) : undefined,
        })),
      }));

      return {
        events,
        pageInfo: eventsData.pageInfo,
        totalCount: eventsData.totalCount,
      };
    },
    [client, envID]
  );
}

export const eventQuery = graphql(`
  query GetEventV2($envID: ID!, $eventID: String!) {
    environment: workspace(id: $envID) {
      eventV2(id: $eventID) {
        name
        id
        receivedAt
        idempotencyKey
        occurredAt
        version
        source
      }
    }
  }
`);

export function useEventDetails() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async ({ eventID }: { eventID: string }) => {
      const result = await client
        .query(
          eventQuery,
          {
            envID,
            eventID,
          },
          { requestPolicy: 'network-only' }
        )
        .toPromise();

      if (result.error) {
        throw new Error(result.error.message);
      }

      if (!result.data) {
        throw new Error('no data returned');
      }

      const eventData = result.data.environment.eventV2;
      return eventData;
    },
    [client, envID]
  );
}

export const eventPayloadQuery = graphql(`
  query GetEventPayload($envID: ID!, $eventID: String!) {
    environment: workspace(id: $envID) {
      eventV2(id: $eventID) {
        raw
      }
    }
  }
`);

export function useEventPayload() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async ({ eventID }: { eventID: string }) => {
      const result = await client
        .query(
          eventPayloadQuery,
          {
            envID,
            eventID,
          },
          { requestPolicy: 'network-only' }
        )
        .toPromise();

      if (result.error) {
        throw new Error(result.error.message);
      }

      if (!result.data) {
        throw new Error('no data returned');
      }

      const eventData = result.data.environment.eventV2;
      return eventData;
    },
    [client, envID]
  );
}
