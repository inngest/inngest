import { createApi } from '@reduxjs/toolkit/query/react';
import { graphqlRequestBaseQuery } from '@rtk-query/graphql-request-base-query';
import { GraphQLClient } from 'graphql-request';

//
// Extend import.meta.env types for type-checking from external packages (e.g., dashboard)
declare global {
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  interface ImportMetaEnv extends Record<string, unknown> {
    readonly VITE_PUBLIC_API_BASE_URL?: string;
  }

  interface ImportMeta {
    readonly env: ImportMetaEnv;
  }
}

const graphQLEndpoint = import.meta.env.VITE_PUBLIC_API_BASE_URL
  ? new URL('/v0/gql', import.meta.env.VITE_PUBLIC_API_BASE_URL)
  : '/v0/gql';

export const client = new GraphQLClient(graphQLEndpoint.toString());

export const api = createApi({
  baseQuery: graphqlRequestBaseQuery({
    // @ts-expect-error
    client,
  }),
  endpoints: () => ({}),
});
