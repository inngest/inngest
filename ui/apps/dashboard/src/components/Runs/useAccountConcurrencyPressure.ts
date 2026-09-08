import { useEffect, useMemo, useState } from 'react';
import { gql, useQuery, type TypedDocumentNode } from 'urql';

import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';
import {
  ACCOUNT_CONCURRENCY_LIMIT_METRIC,
  buildConcurrencyWindow,
  countMinutesWithHits,
  isConcurrencyPressured,
  type ConcurrencyWindow,
} from './accountConcurrency';

type AccountIdentityQuery = {
  account: {
    id: string;
    // Non-null for accounts billed through a provider (Vercel, AWS, ...).
    marketplace: string | null;
    // Null on accounts without a resolved plan; the banner treats that the
    // same as "not yet known" and withholds the billing CTA.
    plan: { isFree: boolean } | null;
  };
};

// Deliberately separate from the metrics query below: this one is cheap (no
// ClickHouse) and always runs, because the dismissal key needs the account ID
// even while the expensive query is skipped. urql shares it with the other
// `account` queries on the page.
//
// Only `plan.isFree` is selected, not the whole plan: the wider GetCurrentPlan
// document also pulls entitlements, which reads the usage table.
const AccountIdentityDocument: TypedDocumentNode<
  AccountIdentityQuery,
  Record<string, never>
> = gql`
  query AccountConcurrencyIdentity {
    account {
      id
      marketplace
      plan {
        isFree
      }
    }
  }
`;

type AccountConcurrencyPressureQuery = {
  // The limit-hit counter is account-scoped, so it is only reachable from the
  // root metrics field — never from workspace().
  concurrencyLimitHits: {
    data: Array<{ bucket: string; value: number }>;
  };
};

type AccountConcurrencyPressureVariables = {
  from: string;
  to: string;
  name: string;
};

const AccountConcurrencyPressureDocument: TypedDocumentNode<
  AccountConcurrencyPressureQuery,
  AccountConcurrencyPressureVariables
> = gql`
  query AccountConcurrencyPressure($from: Time!, $to: Time!, $name: String!) {
    concurrencyLimitHits: metrics(
      opts: { name: $name, from: $from, to: $to }
    ) {
      data {
        bucket
        value
      }
    }
  }
`;

type AccountConcurrencyPressure = {
  isPressured: boolean;
  // How many minutes in the window recorded hits. Reported on the view event so
  // reactions can be read against how strong the signal was.
  minutesWithHits: number;
  // Undefined until the identity query resolves. The banner can't key its
  // dismissal — or render at all — before this is known.
  accountID: string | undefined;
  // Whether the account is billed through a marketplace provider. Assumed
  // false until the identity query resolves; the banner only uses it to
  // suppress a CTA, and the CTA is withheld while the plan is unknown anyway.
  isMarketplace: boolean;
  // Undefined until the identity query resolves, or when the account has no
  // plan attached.
  isFreePlan: boolean | undefined;
};

/**
 * Reports whether the account has repeatedly hit its concurrency limit in the
 * recent past, meaning the queue is holding runs back.
 *
 * Deliberately does NOT poll. The window is resolved once per mount, and again
 * whenever `refreshNonce` changes (the runs page's Refresh action). Polling
 * would make cost scale with how long a tab stays open — a background tab left
 * up all day costs hundreds of ClickHouse reads for a banner nobody is looking
 * at — whereas this scales with page loads. The tradeoff is that pressure
 * starting after load won't raise the banner until the next load or refresh.
 *
 * Pass `skip` when the answer can't change what the caller renders — notably
 * while the banner is dismissed, which suppresses the query entirely.
 */
export function useAccountConcurrencyPressure({
  refreshNonce = 0,
  skip = false,
}: { refreshNonce?: number; skip?: boolean } = {}): AccountConcurrencyPressure {
  const [{ data: identity }] = useQuery({ query: AccountIdentityDocument });

  const accountID = identity?.account.id;

  // Resolved in an effect rather than during render: this component is
  // server-rendered, and a render-time Date.now() would give the server and
  // client different query variables (and so different cache keys) on hydrate.
  const [window, setWindow] = useState<ConcurrencyWindow | null>(null);

  useEffect(() => {
    if (skip) {
      return;
    }
    setWindow(buildConcurrencyWindow(Date.now()));
  }, [refreshNonce, skip]);

  const variables = useMemo(
    () => ({
      from: window?.from ?? '',
      to: window?.to ?? '',
      name: ACCOUNT_CONCURRENCY_LIMIT_METRIC,
    }),
    [window],
  );

  const { data } = useSkippableGraphQLQuery({
    query: AccountConcurrencyPressureDocument,
    variables,
    skip: skip || !window,
  });

  return {
    accountID,
    isFreePlan: identity?.account.plan?.isFree,
    isMarketplace: Boolean(identity?.account.marketplace),
    isPressured: isConcurrencyPressured(data?.concurrencyLimitHits.data),
    minutesWithHits: countMinutesWithHits(data?.concurrencyLimitHits.data),
  };
}
