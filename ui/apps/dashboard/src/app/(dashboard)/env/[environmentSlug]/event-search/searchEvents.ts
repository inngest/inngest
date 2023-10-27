import { Client } from 'urql';

import { graphql } from '@/gql';
import type { EventSearchFilterField } from '@/gql/graphql';
import type { Event } from './types';

const query = graphql(`
  query SearchEvents(
    $environmentID: ID!
    $fields: [EventSearchFilterField!]!
    $lowerTime: Time!
    $upperTime: Time!
  ) {
    environment: workspace(id: $environmentID) {
      id
      eventSearch(filter: { fields: $fields, lowerTime: $lowerTime, upperTime: $upperTime }) {
        edges {
          node {
            id
            name
            receivedAt
          }
        }
        pageInfo {
          hasNextPage
          hasPreviousPage
          startCursor
          endCursor
        }
      }
    }
  }
`);

type Filter = {
  fields: EventSearchFilterField[];
  lowerTime: Date;
  upperTime: Date;
};

export async function searchEvents({
  client,
  environmentID,
  fields,
  lowerTime,
  upperTime,
}: { client: Client; environmentID: string } & Filter): Promise<Event[]> {
  const res = await client
    .query(query, {
      environmentID,
      fields,
      lowerTime: lowerTime.toISOString(),
      upperTime: upperTime.toISOString(),
    })
    .toPromise();

  if (res.error) {
    throw res.error;
  }

  const edges = res.data?.environment.eventSearch.edges;
  if (!edges) {
    // Should be unreachable.
    throw new Error('finished fetching but missing data');
  }

  const data: Event[] = [];
  for (const edge of edges) {
    if (!edge) {
      continue;
    }

    data.push({
      ...edge.node,
      receivedAt: new Date(edge.node.receivedAt),
    });
  }

  return data;
}
