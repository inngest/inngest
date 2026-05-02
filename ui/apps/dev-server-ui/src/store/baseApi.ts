import { createApi } from '@reduxjs/toolkit/query/react';
import type { BaseQueryFn } from '@reduxjs/toolkit/query';
import { graphqlRequestBaseQuery } from '@rtk-query/graphql-request-base-query';
import { GraphQLClient } from 'graphql-request';

const graphQLEndpoint = import.meta.env.VITE_PUBLIC_API_BASE_URL
  ? new URL('/v0/gql', import.meta.env.VITE_PUBLIC_API_BASE_URL)
  : '/v0/gql';

export const client = new GraphQLClient(graphQLEndpoint.toString(), {
  credentials: 'include',
});

const rawBaseQuery = graphqlRequestBaseQuery({
  // @ts-expect-error
  client,
});

const baseQueryWithReauth: BaseQueryFn = async (args, api, extraOptions) => {
  const result = await rawBaseQuery(args, api, extraOptions);
  if (
    result.error &&
    'originalStatus' in (result.error as any) &&
    (result.error as any).originalStatus === 401
  ) {
    window.location.href = '/login';
  }
  return result;
};

export const api = createApi({
  baseQuery: baseQueryWithReauth,
  endpoints: () => ({}),
});
