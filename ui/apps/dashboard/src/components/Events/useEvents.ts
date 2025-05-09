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

// TODO: Replace with real API
export function useEventDetails() {
  return useCallback(async ({ eventName }: { eventName: string }) => {
    console.log(eventName);
    await new Promise((resolve) => setTimeout(resolve, 500));

    const event = {
      id: '01JGPM6FYSRN9C0ZGJ7PXPVRGY',
      receivedAt: new Date('2025-04-10T16:43:21.696Z'),
      idempotencyKey: 'custom-payload-id',
      name: 'UserSignedUp',
      source: 'Default Inngest key',
      timestamp: new Date(1745226902417),
      version: '2022-12-16',
    };

    return event;
  }, []);
}

export function useEventPayload() {
  return useCallback(async ({ eventName }: { eventName: string }) => {
    console.log(eventName);
    await new Promise((resolve) => setTimeout(resolve, 500));

    const event = {
      name: 'UserSignedUp',
      payload:
        '{\n  "name": "signup.new",\n  "data": {\n    "account_id": "119f5971-9878-46bd-a18f-4fecd",\n    "method": "",\n    "plan_name": "Free Tier"\n  },\n  "id": "119f5971-9878-46bd-a18f-4f0680174ecd",\n  "ts": 1711051784369,\n  "v": "2021-05-11.01"\n}',
    };

    return event;
  }, []);
}
