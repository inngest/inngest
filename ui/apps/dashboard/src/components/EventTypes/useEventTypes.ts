import { useCallback } from 'react';
import { getTimestampDaysAgo } from '@inngest/components/utils/date';
import { useQuery } from '@tanstack/react-query';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

// prettier-ignore
const query = graphql(`
  query GetEventTypesV2($envID: ID!, $cursor: String, $archived: Boolean, $nameSearch: String) {
    environment: workspace(id: $envID) {
      eventTypesV2(
        after: $cursor
        first: 40
        filter: { archived: $archived, nameSearch: $nameSearch }
      ) {
        edges {
          node {
            name
            functions {
              edges {
                node {
                  id
                  slug
                  name
                }
              }
            }
          }
        }
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

type QueryVariables = {
  archived: boolean;
  nameSearch: string | null;
  cursor: string | null;
};

export function useEventTypes() {
  const envID = useEnvironment().id;
  const client = useClient();
  return useCallback(
    async ({ cursor, archived, nameSearch }: QueryVariables) => {
      const result = await client
        .query(
          query,
          {
            envID,
            archived,
            cursor,
            nameSearch,
          },
          { requestPolicy: 'network-only' },
        )
        .toPromise();

      if (result.error) {
        throw new Error(result.error.message);
      }

      if (!result.data) {
        throw new Error('no data returned');
      }

      const eventTypesData = result.data.environment.eventTypesV2;
      const events = eventTypesData.edges.map(({ node }) => ({
        name: node.name,
        functions: node.functions.edges.map((f) => f.node),
        archived,
      }));

      return {
        events,
        pageInfo: eventTypesData.pageInfo,
      };
    },
    [client, envID],
  );
}

type VolumeQueryVariables = {
  eventName: string;
};

// prettier-ignore
const volumeQuery = graphql(`
  query GetEventTypeVolumeV2($envID: ID!, $eventName: String!, $startTime: Time!, $endTime: Time!) {
    environment: workspace(id: $envID) {
      eventType(name: $eventName) {
        name
        usage(opts: { period: hour, from: $startTime, to: $endTime }) {
          total
          data {
            count
            slot
          }
        }
      }
    }
  }
`);

export function useEventTypeVolume() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async ({ eventName }: VolumeQueryVariables) => {
      const startTime = getTimestampDaysAgo({
        currentDate: new Date(),
        days: 1,
      }).toISOString();
      const endTime = new Date().toISOString();
      const result = await client
        .query(
          volumeQuery,
          {
            envID,
            eventName,
            startTime,
            endTime,
          },
          { requestPolicy: 'network-only' },
        )
        .toPromise();

      if (result.error) {
        throw new Error(result.error.message);
      }

      if (!result.data) {
        throw new Error('no data returned');
      }

      const eventType = result.data.environment.eventType;

      const dailyVolumeSlots = eventType.usage.data.map((slot) => ({
        startCount: slot.count,
        slot: slot.slot,
      }));

      return {
        name: eventType.name,
        volume: {
          totalVolume: eventType.usage.total,
          dailyVolumeSlots,
        },
      };
    },
    [client, envID],
  );
}

// prettier-ignore
const eventTypeQuery = graphql(`
  query GetEventType($envID: ID!, $eventName: String!) {
    environment: workspace(id: $envID) {
      eventType(name: $eventName) {
        name
        functions {
          edges {
            node {
              id
              slug
              name
            }
          }
        }
      }
    }
  }
`);

export function useEventType({ eventName }: { eventName: string }) {
  const envID = useEnvironment().id;
  const client = useClient();

  return useQuery({
    queryKey: ['event-type', envID, eventName],
    queryFn: async () => {
      const result = await client
        .query(eventTypeQuery, { envID, eventName })
        .toPromise();

      if (result.error) {
        throw result.error;
      }

      const eventType = result.data?.environment.eventType;

      if (!eventType) {
        return null;
      }

      return {
        ...eventType,
        functions: eventType.functions.edges.map(({ node }) => node),
      };
    },
  });
}

// prettier-ignore
export const allEventTypesQuery = graphql(`
  query GetAllEventNames($envID: ID!) {
    environment: workspace(id: $envID) {
      eventTypesV2(first: 40, filter: {}) {
        edges {
          node {
            name
          }
        }
      }
    }
  }
`);

export function useAllEventTypes() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(async () => {
    const result = await client
      .query(allEventTypesQuery, { envID }, { requestPolicy: 'network-only' })
      .toPromise();

    if (result.error) {
      throw new Error(result.error.message);
    }

    if (!result.data) {
      throw new Error('no data returned');
    }

    const eventsData = result.data.environment.eventTypesV2;
    const events = eventsData.edges.map(({ node }) => ({
      id: node.name,
      name: node.name,
      latestSchema: '',
    }));

    return events;
  }, [client, envID]);
}
