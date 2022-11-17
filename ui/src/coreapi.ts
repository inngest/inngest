import { gql } from "graphql-request";

export const EVENTS_STREAM = gql`
  query GetEventsStream($query: EventsQuery! = {}) {
    events(query: $query) {
      id
      name
      createdAt
      status
      pendingRuns
    }
  }
`;
