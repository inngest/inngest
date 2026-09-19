import { graphql } from '@/gql';

export const AppFilterDocument = graphql(`
  query AppFilter($envSlug: String!) {
    env: envBySlug(slug: $envSlug) {
      apps {
        externalID
        id
        name
      }
    }
  }
`);
