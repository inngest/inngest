import { gql, useQuery, type TypedDocumentNode } from 'urql';

import {
  alertState,
  concurrencyLinkProps,
  effectiveLimit,
  usedRatio,
  type ConcurrencyAlertState,
  type ConcurrencyAlertSurface,
} from './concurrencyAlert';

export type ConcurrencyAlertCheckQuery = {
  account: {
    id: string;
    marketplace: string | null;
    plan: { isFree: boolean } | null;
    burstConcurrency: {
      limit: number;
      budgetMinutes: number;
      bursting: boolean;
    } | null;
    concurrencyStatus: {
      limitReached: boolean;
      minutesWithHits: number;
    } | null;
    entitlements: { concurrency: { limit: number } };
  };
};

export const ConcurrencyAlertCheckDocument: TypedDocumentNode<
  ConcurrencyAlertCheckQuery,
  Record<string, never>
> = gql`
  query ConcurrencyAlertCheck {
    account {
      id
      marketplace
      plan {
        isFree
      }
      burstConcurrency {
        limit
        budgetMinutes
        bursting
      }
      concurrencyStatus {
        limitReached
        minutesWithHits
      }
      entitlements: ents {
        concurrency {
          limit
        }
      }
    }
  }
`;

type ConcurrencyBurstUsageQuery = {
  account: {
    burstConcurrency: { usage: { minutesRemaining: number } } | null;
  };
};

export const ConcurrencyBurstUsageDocument: TypedDocumentNode<
  ConcurrencyBurstUsageQuery,
  Record<string, never>
> = gql`
  query ConcurrencyBurstUsage {
    account {
      burstConcurrency {
        usage {
          minutesRemaining
        }
      }
    }
  }
`;

export type ConcurrencyAlert = {
  state: ConcurrencyAlertState;
  limit: number | undefined;
  minutesRemaining: number | undefined;
  usedRatio: number;
  linkProps: (surface: ConcurrencyAlertSurface) => { to: string };
};

export function useConcurrencyAlert(): ConcurrencyAlert | null {
  const [{ data }] = useQuery({
    query: ConcurrencyAlertCheckDocument,
    requestPolicy: 'cache-first',
  });

  const burst = data?.account.burstConcurrency;
  const bursting = burst?.bursting ?? false;
  const [{ data: usageData }] = useQuery({
    query: ConcurrencyBurstUsageDocument,
    requestPolicy: 'cache-first',
    pause: !bursting,
  });

  const state = alertState({
    boosted: bursting,
    pressured: data?.account.concurrencyStatus?.limitReached ?? false,
  });
  if (!state) {
    return null;
  }

  const minutesRemaining =
    usageData?.account.burstConcurrency?.usage.minutesRemaining;

  const limit = effectiveLimit({
    state,
    burstLimit: burst?.limit,
    baseLimit: data?.account.entitlements.concurrency.limit,
  });

  return {
    state,
    limit,
    minutesRemaining,
    usedRatio:
      burst && minutesRemaining !== undefined
        ? usedRatio({ budgetMinutes: burst.budgetMinutes, minutesRemaining })
        : 0,
    linkProps: (surface) => concurrencyLinkProps({ state, surface }),
  };
}
