import { Client } from 'urql';

import { graphql } from '@/gql';
import type { Event } from './types';

const queryDoc = graphql(`
  query SearchEvents($environmentID: ID!, $lowerTime: Time!, $query: String!, $upperTime: Time!) {
    environment: workspace(id: $environmentID) {
      id
      eventSearch(filter: { lowerTime: $lowerTime, query: $query, upperTime: $upperTime }) {
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
  query: string;
  upperTime: Date;
};

export async function searchEvents({
  client,
  environmentID,
  lowerTime,
  query,
  upperTime,
}: { client: Client; environmentID: string } & Filter): Promise<Event[]> {
  const res = await client
    .query(queryDoc, {
      environmentID,
      lowerTime: lowerTime.toISOString(),
      query,
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
