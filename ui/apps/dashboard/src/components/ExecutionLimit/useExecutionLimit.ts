import { graphql } from '@/gql';
import { pathCreator } from '@/utils/urls';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const executionLimitQuery = graphql(`
  query ExecutionLimitCheck {
    account {
      id
      marketplaceBillingURL
      plan {
        id
        name
      }
      entitlements: ents {
        executions {
          limit
          overageAllowed
        }
        usage {
          executions
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
  const res = useGraphQLQuery({ query: executionLimitQuery, variables: {} });
  if (!res.data) return null;

  const { limit, overageAllowed } = res.data.account.entitlements.executions;
  const usage = res.data.account.entitlements.usage.executions;
  if (limit === null) return null;

  const isEnterprise = (res.data.account.plan?.name ?? '')
    .toLowerCase()
    .includes('enterprise');

  return {
    isCapped: !isEnterprise && !overageAllowed && usage >= limit,
    usedExecutions: usage,
    executionLimit: limit,
    marketplaceBillingURL: res.data.account.marketplaceBillingURL ?? null,
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
