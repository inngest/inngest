import { createApi } from '@reduxjs/toolkit/query/react';
import { graphqlRequestBaseQuery } from '@rtk-query/graphql-request-base-query';
import { GraphQLClient } from 'graphql-request';

const graphQLEndpoint = process.env.NEXT_PUBLIC_API_BASE_URL
  ? new URL('/v0/gql', process.env.NEXT_PUBLIC_API_BASE_URL)
  : '/v0/gql';

export const client = new GraphQLClient(graphQLEndpoint.toString());

export const api = createApi({
  // @ts-expect-error
  baseQuery: graphqlRequestBaseQuery({ client }),
  endpoints: () => ({}),
});
