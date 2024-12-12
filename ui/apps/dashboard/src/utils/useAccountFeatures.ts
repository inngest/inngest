import type { Features } from '@inngest/components/types/features';

import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query GetAccountEntitlements {
    account {
      entitlements {
        history {
          limit
        }
      }
    }
  }
`);

export function useAccountFeatures() {
  const res = useGraphQLQuery({ query, variables: {} });

  if (res.data) {
    const features: Features = {
      history: res.data.account.entitlements.history.limit || 7,
    };

    return {
      ...res,
      data: features,
    };
  }

  return res;
}
