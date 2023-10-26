import { Client } from 'urql';

import { graphql } from '@/gql';
import type { Event } from './types';

const query = graphql(`
  query SearchEvents($environmentID: ID!, $lowerTime: Time!, $text: String!, $upperTime: Time!) {
    environment: workspace(id: $environmentID) {
      id
      eventSearch(filter: { lowerTime: $lowerTime, text: $text, upperTime: $upperTime }) {
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
  lowerTime: Date;
  text: string;
  upperTime: Date;
};

export async function searchEvents({
  client,
  environmentID,
  lowerTime,
  text,
  upperTime,
}: { client: Client; environmentID: string } & Filter): Promise<Event[]> {
  const res = await client
    .query(query, {
      environmentID,
      lowerTime: lowerTime.toISOString(),
      text,
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
