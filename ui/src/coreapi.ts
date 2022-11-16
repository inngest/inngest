import { ApolloClient, InMemoryCache } from "@apollo/client";
import { graphql } from "./gql";

export const client = new ApolloClient({
  uri: "http://localhost:4000/graphql",
  cache: new InMemoryCache(),
  name: "Dev Server Core API",
});

export const EVENTS_STREAM = graphql(`
  query GetEventsStream($query: EventsQuery!) {
    events(query: $query) {
      id
      name
      createdAt
      payload
    }
  }
`);
