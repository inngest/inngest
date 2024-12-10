import type { Features } from '@inngest/components/types/features';

import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query GetPlanEntitlements {
    account {
      plan {
        entitlements {
          history {
            limit
          }
        }
      }
    }
  }
`);

export function usePlanFeatures() {
  const res = useGraphQLQuery({ query, variables: {} });

  if (res.data) {
    const features: Features = {
      history: res.data.account.plan?.entitlements.history.limit || 7,
    };

    return {
      ...res,
      data: features,
    };
  }

  return res;
}
