import 'server-only';
import { graphql } from '@/gql';
import { type EntitlementUsageQuery } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';

export const entitlementUsageDocument = graphql(`
  query EntitlementUsage {
    account {
      id
      entitlementUsage {
        runCount {
          current
          limit
        }
      }
    }
  }
`);

export const entitlementUsage = async () => {
  return await graphqlAPI.request<EntitlementUsageQuery>(entitlementUsageDocument);
};
