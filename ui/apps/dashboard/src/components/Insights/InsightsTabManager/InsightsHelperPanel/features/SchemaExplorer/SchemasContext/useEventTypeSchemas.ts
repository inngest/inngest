import { useCallback } from 'react';
import type { PageInfo } from '@inngest/components/types/eventType';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

const GET_EVENT_TYPE_SCHEMAS_QUERY = graphql(`
  query GetEventTypeSchemas($envID: ID!, $cursor: String, $nameSearch: String, $archived: Boolean) {
    environment: workspace(id: $envID) {
      eventTypesV2(
        after: $cursor
        first: 40
        filter: { archived: $archived, nameSearch: $nameSearch }
      ) {
        edges {
          node {
            name
            latestSchema
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
  nameSearch: string | null;
  cursor: string | null;
};

export function useEventTypeSchemas() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async ({
      cursor,
      nameSearch,
    }: QueryVariables): Promise<{
      events: { name: string; schema: string }[];
      pageInfo: PageInfo;
    }> => {
      const result = await client
        .query(
          GET_EVENT_TYPE_SCHEMAS_QUERY,
          { archived: false, cursor, envID, nameSearch },
          { requestPolicy: 'network-only' },
        )
        .toPromise();

      if (result.error) throw new Error(result.error.message);
      if (!result.data) throw new Error('no data returned');

      const eventTypesData = result.data.environment.eventTypesV2;
      const events = eventTypesData.edges.map(({ node }) => ({
        name: node.name,
        schema: node.latestSchema ?? '',
      }));

      return { events, pageInfo: eventTypesData.pageInfo };
    },
    [client, envID],
  );
}
