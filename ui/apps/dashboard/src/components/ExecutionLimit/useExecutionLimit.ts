import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { graphql } from '@/gql';
import { pathCreator } from '@/utils/urls';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const executionLimitQuery = graphql(`
  query ExecutionLimitCheck {
    account {
      id
      marketplaceBillingURL
      entitlements {
        executions {
          usage
          limit
          overageAllowed
        }
      }
    }
  }
`);

type ExecutionLimitData = {
  isCapped: boolean;
  usedExecutions: number;
  executionLimit: number;
  marketplaceBillingURL: string | null;
};

export function useExecutionLimit(): ExecutionLimitData | null {
  const { value: deepLinkingEnabled } = useBooleanFlag('vercel-deep-linking');

  const res = useGraphQLQuery({ query: executionLimitQuery, variables: {} });
  if (!res.data) return null;

  const { usage, limit, overageAllowed } =
    res.data.account.entitlements.executions;
  if (limit === null) return null;

  return {
    isCapped: !overageAllowed && usage >= limit,
    usedExecutions: usage,
    executionLimit: limit,
    marketplaceBillingURL: deepLinkingEnabled
      ? res.data.account.marketplaceBillingURL ?? null
      : null,
  };
}

// Marketplace-billed accounts are bounced from every /billing route but usage,
// so they upgrade on their provider's page. Both branches must travel as `to`:
// TanStack's useLinkProps overwrites `href` with a location it builds itself.
export function upgradeLinkProps(
  marketplaceBillingURL: string | null,
  ref: string,
) {
  return {
    to: marketplaceBillingURL ?? pathCreator.billing({ tab: 'plans', ref }),
  };
}
