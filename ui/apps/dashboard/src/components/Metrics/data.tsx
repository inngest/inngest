import 'server-only';
import { cache } from 'react';

import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';

const MetricsLookupDocument = graphql(`
  query MetricsLookups($envSlug: String!) {
    envBySlug(slug: $envSlug) {
      apps {
        externalID
        id
        name
      }
      workflows {
        data {
          name
          id
          slug
        }
      }
    }
  }
`);

export const preloadMetricsLookups = (envSlug: string) => {
  void getMetricsLookups(envSlug);
};

export const getMetricsLookups = cache(async (envSlug: string) => {
  return await graphqlAPI.request<{ envBySlug: { apps: []; workflows: { data: [] } } }>(
    MetricsLookupDocument,
    { envSlug }
  );
});
