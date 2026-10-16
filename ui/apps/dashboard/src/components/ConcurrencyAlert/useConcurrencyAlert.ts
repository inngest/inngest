import { gql, useQuery, type TypedDocumentNode } from 'urql';

import {
  alertState,
  concurrencyLinkProps,
  effectiveLimit,
  usedRatio,
  type ConcurrencyAlertState,
  type ConcurrencyAlertSurface,
} from './concurrencyAlert';

type ConcurrencyAlertCheckQuery = {
  account: {
    id: string;
    marketplace: string | null;
    marketplaceBillingURL: string | null;
    plan: { isFree: boolean } | null;
    burstConcurrency: {
      limit: number;
      budgetMinutes: number;
      usage: {
        minutesRemaining: number;
        bursting: boolean;
      };
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
      marketplaceBillingURL
      plan {
        isFree
      }
      burstConcurrency {
        limit
        budgetMinutes
        usage {
          minutesRemaining
          bursting
        }
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
  const state = alertState({
    boosted: burst?.usage.bursting ?? false,
    pressured: data?.account.concurrencyStatus?.limitReached ?? false,
  });
  if (!state) {
    return null;
  }

  const limit = effectiveLimit({
    state,
    burstLimit: burst?.limit,
    baseLimit: data?.account.entitlements.concurrency.limit,
  });

  return {
    state,
    limit,
    minutesRemaining: burst?.usage.minutesRemaining,
    usedRatio: burst
      ? usedRatio({
          budgetMinutes: burst.budgetMinutes,
          minutesRemaining: burst.usage.minutesRemaining,
        })
      : 0,
    linkProps: (surface) =>
      concurrencyLinkProps({
        state,
        surface,
        isMarketplace: Boolean(data?.account.marketplace),
        isFreePlan: data?.account.plan?.isFree,
        marketplaceBillingURL: data?.account.marketplaceBillingURL,
      }),
  };
}
