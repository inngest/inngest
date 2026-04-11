import { createApi } from '@reduxjs/toolkit/query/react';
import { graphqlRequestBaseQuery } from '@rtk-query/graphql-request-base-query';
import { GraphQLClient } from 'graphql-request';

const baseOrigin =
  import.meta.env.VITE_PUBLIC_API_BASE_URL ||
  (typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8288');
const graphQLEndpoint = new URL('/v0/gql', baseOrigin);

export const client = new GraphQLClient(graphQLEndpoint.toString());

export const api = createApi({
  baseQuery: graphqlRequestBaseQuery({
    client,
  }),
  endpoints: () => ({}),
});
