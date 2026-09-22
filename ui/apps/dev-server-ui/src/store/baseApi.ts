import { createApi } from '@reduxjs/toolkit/query/react';
import { graphqlRequestBaseQuery } from '@rtk-query/graphql-request-base-query';
import { GraphQLClient } from 'graphql-request';
import { createDevServerURL } from '@/utils/devServer';

// the shared helper keeps GraphQL and REST on the same server.  hosted builds
// use localhost and the selected port.  local builds use VITE_PUBLIC_API_BASE_URL or same-origin paths.
const graphQLEndpoint = createDevServerURL('/v0/gql');

export const client = new GraphQLClient(graphQLEndpoint);

export const api = createApi({
  baseQuery: graphqlRequestBaseQuery({
    // @ts-expect-error
    client,
  }),
  endpoints: () => ({}),
});
