import { useCallback } from 'react';
import { getTimestampDaysAgo } from '@inngest/components/utils/date';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

const query = graphql(`
  query GetEventTypesV2($envID: ID!, $cursor: String, $archived: Boolean) {
    environment: workspace(id: $envID) {
      eventTypesV2(after: $cursor, first: 30, filter: { archived: $archived }) {
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
  cursor: string | null;
};

export function useEventTypes() {
  const envID = useEnvironment().id;
  const client = useClient();
  return useCallback(
    async ({ cursor, archived }: QueryVariables) => {
      const result = await client
        .query(
          query,
          {
            envID,
            archived,
            cursor,
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
    [client, envID]
  );
}

type VolumeQueryVariables = {
  archived: boolean;
  cursor: string | null;
};

const volumeQuery = graphql(`
  query GetEventTypesVolumeV2(
    $envID: ID!
    $cursor: String
    $archived: Boolean
    $startTime: Time!
    $endTime: Time!
  ) {
    environment: workspace(id: $envID) {
      eventTypesV2(after: $cursor, first: 30, filter: { archived: $archived }) {
        edges {
          node {
            name
            usage(opts: { period: day, from: $startTime, to: $endTime }) {
              total
              data {
                count
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

export function useEventTypesVolume() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async ({ cursor, archived }: VolumeQueryVariables) => {
      const startTime = getTimestampDaysAgo({ currentDate: new Date(), days: 1 }).toISOString();
      const endTime = new Date().toISOString();
      const result = await client
        .query(
          volumeQuery,
          {
            envID,
            archived,
            cursor,
            startTime,
            endTime,
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

      const eventTypes = result.data.environment.eventTypesV2.edges;

      const events = eventTypes.map(({ node }) => {
        const dailyVolumeSlots = node.usage.data.map((slot) => ({
          startCount: slot.count,
        }));

        return {
          name: node.name,
          volume: {
            totalVolume: node.usage.total,
            dailyVolumeSlots,
          },
        };
      });

      return {
        events,
        pageInfo: result.data.environment.eventTypesV2.pageInfo,
      };
    },
    [client, envID]
  );
}
