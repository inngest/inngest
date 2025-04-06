import { useCallback } from 'react';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

const query = graphql(`
  query GetNewEventTypes($envID: ID!) {
    environment: workspace(id: $envID) {
      events {
        data {
          name
          functions: workflows {
            id
            slug
            name
          }
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

      const eventTypesData = result.data.environment.events.data;
      const events = eventTypesData.map((event) => {
        return {
          // TODO: fetch archived
          archived: false,
          ...event,
        };
      });

      return {
        events: events,
        // TODO: add pagination to API
        pageInfo: {
          hasNextPage: false,
          hasPreviousPage: false,
          endCursor: null,
          startCursor: null,
        },
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
  query GetNewEventTypesVolume($envID: ID!) {
    environment: workspace(id: $envID) {
      events {
        data {
          name
          dailyVolume: usage(opts: { period: "hour", range: "day" }) {
            total
            data {
              count
            }
          }
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
      const result = await client
        .query(
          volumeQuery,
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

      const eventTypes = result.data.environment.events.data;

      const events = eventTypes.map((event) => {
        const dailyVolumeSlots = event.dailyVolume.data.map((slot) => ({
          startCount: slot.count,
        }));

        return {
          name: event.name,
          volume: {
            totalVolume: event.dailyVolume.total,
            dailyVolumeSlots: dailyVolumeSlots,
          },
        };
      });

      return {
        events,
        pageInfo: {
          hasNextPage: false, // TODO: Update when pagination is supported
          hasPreviousPage: false,
          endCursor: null,
          startCursor: null,
        },
      };
    },
    [client, envID]
  );
}
