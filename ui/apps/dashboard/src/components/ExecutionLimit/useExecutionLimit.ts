import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { graphql } from '@/gql';
import { Marketplace } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';

import { isExecutionCapped, legacyExecutionCap } from './executionLimit';

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
        overageAllowed
      }
    }
  }
`);

type ExecutionLimitData = {
  isCapped: boolean;
  usedExecutions: number;
  executionLimit: number;
  isVercel: boolean;
  marketplaceBillingURL: string | null;
  enhanced: boolean;
};

export function useExecutionLimit(): ExecutionLimitData | null {
  const { value: deepLinkingEnabled } = useBooleanFlag('vercel-deep-linking');
  const { value: enhanced, isReady } = useBooleanFlag(
    'hobby-execution-limit-ui',
  );

  const legacyRes = useSkippableGraphQLQuery({
    query: executionLimitQuery,
    variables: {},
    skip: !isReady || enhanced,
  });
  const capRes = useSkippableGraphQLQuery({
    query: executionCapQuery,
    variables: {},
    skip: !isReady || !enhanced,
  });

  const account = enhanced ? capRes.data?.account : legacyRes.data?.account;
  if (!account) return null;

  const cap =
    'executionCap' in account
      ? account.executionCap
      : legacyExecutionCap(account.entitlements.executions);
  if (!cap) return null;

  return {
    isCapped: isExecutionCapped(cap),
    usedExecutions: cap.usage,
    executionLimit: cap.limit,
    isVercel:
      'marketplace' in account && account.marketplace === Marketplace.Vercel,
    marketplaceBillingURL: deepLinkingEnabled
      ? account.marketplaceBillingURL ?? null
      : null,
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
