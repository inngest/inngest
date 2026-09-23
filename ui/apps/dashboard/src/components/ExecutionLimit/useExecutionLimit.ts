import {
  useBooleanFlag,
  useIsIdentificationSettled,
} from '@/components/FeatureFlags/hooks';
import { graphql } from '@/gql';
import { Marketplace } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';

import {
  isExecutionCapped,
  legacyExecutionCap,
  usageBand,
  type UsageBand,
} from './executionLimit';

const executionLimitQuery = graphql(`
  query ExecutionLimitCheck {
    account {
      id
      marketplaceBillingURL
      plan {
        id
        name
      }
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

const executionCapQuery = graphql(`
  query ExecutionCapCheck {
    account {
      id
      marketplace
      marketplaceBillingURL
      executionCap {
        usage
        limit
        enforced
        exceeded
      }
    }
  }
`);

type ExecutionLimitData = {
  accountID: string;
  band: UsageBand;
  isCapped: boolean;
  usedExecutions: number;
  executionLimit: number;
  isVercel: boolean;
  marketplaceBillingURL: string | null;
  enhanced: boolean;
};

export function useExecutionLimit(): ExecutionLimitData | null {
  const { value: enhancedEnabled, isReady } = useBooleanFlag(
    'hobby-execution-limit-ui',
  );
  const enhanced = isReady && enhancedEnabled;

  // Gates both queries so a flagged account never renders the legacy card
  // first, without stranding accounts that never identify.
  const isSettled = useIsIdentificationSettled();

  const legacyRes = useSkippableGraphQLQuery({
    query: executionLimitQuery,
    variables: {},
    skip: !isSettled || enhanced,
  });
  const capRes = useSkippableGraphQLQuery({
    query: executionCapQuery,
    variables: {},
    skip: !isSettled || !enhanced,
  });

  const account = enhanced ? capRes.data?.account : legacyRes.data?.account;
  if (!account) return null;

  const cap =
    'executionCap' in account
      ? account.executionCap
      : legacyExecutionCap(account.entitlements.executions);
  if (!cap) return null;

  const isEnterprise =
    'plan' in account &&
    (account.plan?.name ?? '').toLowerCase().includes('enterprise');
  const isCapped = !isEnterprise && isExecutionCapped(cap);

  return {
    accountID: account.id,
    band: usageBand({ usage: cap.usage, limit: cap.limit, isCapped }),
    isCapped,
    usedExecutions: cap.usage,
    executionLimit: cap.limit,
    isVercel:
      'marketplace' in account && account.marketplace === Marketplace.Vercel,
    marketplaceBillingURL: account.marketplaceBillingURL ?? null,
    enhanced,
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
