import { createApi } from '@reduxjs/toolkit/query/react';
import { graphqlRequestBaseQuery } from '@rtk-query/graphql-request-base-query';
import { GraphQLClient } from 'graphql-request';

//
// Extend import.meta.env types for type-checking from external packages (e.g., dashboard)
// FYI this is here because we actually import from the dev server ui in
// ui/apps/dev-server-ui/src/components/Function/FunctionConfigurationContainer.tsx
// This causes to reach out here via generated code which fails type checking without this.
// TODO: this can go away when dashboard is migrated.
declare global {
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  interface ImportMetaEnv extends Record<string, unknown> {
    readonly VITE_PUBLIC_API_BASE_URL?: string;
  }

  interface ImportMeta {
    readonly env: ImportMetaEnv;
  }
}

//
// TODO: temporary hack here until we are completely cut over to tanstack
// since this code actually gets evaluated in dashboard (see above)
const viteEnv = import.meta.env;

const graphQLEndpoint = viteEnv?.VITE_PUBLIC_API_BASE_URL
  ? new URL('/v0/gql', viteEnv.VITE_PUBLIC_API_BASE_URL)
  : process.env.NEXT_PUBLIC_API_BASE_URL
  ? new URL('/v0/gql', process.env.NEXT_PUBLIC_API_BASE_URL)
  : '/v0/gql';

export const client = new GraphQLClient(graphQLEndpoint.toString());

export const api = createApi({
  baseQuery: graphqlRequestBaseQuery({
    // @ts-expect-error
    client,
  }),
  endpoints: () => ({}),
});
