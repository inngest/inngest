'use client';

import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

export const vercelIntegrationQuery = graphql(`
  query VercelIntegration {
    account {
      vercelIntegration {
        isMarketplace
        projects {
          canChangeEnabled
          deploymentProtection
          isEnabled
          name
          originOverride
          projectID
          protectionBypassSecret
          servePath
        }
      }
    }
  }
`);

export function useVercelIntegration() {
  const res = useGraphQLQuery({
    query: vercelIntegrationQuery,
    variables: {},
  });

  if (!res.data) {
    return res;
  }

  return {
    ...res,
    data: res.data.account.vercelIntegration,
  };
}
