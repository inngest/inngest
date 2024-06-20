import type { Features } from '@inngest/components/types/features';

import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query GetPlanFeatures {
    account {
      plan {
        features
      }
    }
  }
`);

export function usePlanFeatures() {
  const res = useGraphQLQuery({ query, variables: {} });

  if (res.data) {
    const features: Features = {
      history: 7,
    };

    const rawHistory = res.data.account?.plan?.features?.log_retention;
    if (typeof rawHistory === 'number') {
      features.history = rawHistory;
    }

    return {
      ...res,
      data: features,
    };
  }

  return res;
}
