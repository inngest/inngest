import { useEffect, useState } from 'react';
import { gql, useQuery, type TypedDocumentNode } from 'urql';

import { useAccountConcurrencyPressure } from '../Runs/useAccountConcurrencyPressure';
import {
  alertState,
  concurrencyLinkProps,
  effectiveLimit,
  isBurstActive,
  usedRatio,
  type ConcurrencyAlertState,
  type ConcurrencyAlertSurface,
} from './concurrencyAlert';

type ConcurrencyAlertCheckQuery = {
  account: {
    marketplaceBillingURL: string | null;
    burstConcurrency: {
      limit: number;
      budgetMinutes: number;
      exhaustedAt: string | null;
      usage: {
        minutesRemaining: number;
        bursts: Array<{ endedAt: string }>;
      };
    } | null;
    entitlements: { concurrency: { limit: number } };
  };
};

const ConcurrencyAlertCheckDocument: TypedDocumentNode<
  ConcurrencyAlertCheckQuery,
  Record<string, never>
> = gql`
  query ConcurrencyAlertCheck {
    account {
      marketplaceBillingURL
      burstConcurrency {
        limit
        budgetMinutes
        exhaustedAt
        usage {
          minutesRemaining
          bursts {
            endedAt
          }
        }
      }
      entitlements: ents {
        concurrency {
          limit
        }
      }
    }
  }
`;

// cache-first means every mount shares one result, so each result keeps the
// time it first arrived. Otherwise a later mount would judge old bursts
// against a newer clock and disagree with the pill.
const receivedAt = new WeakMap<object, number>();

export type ConcurrencyAlert = {
  state: ConcurrencyAlertState;
  limit: number | undefined;
  minutesRemaining: number | undefined;
  usedRatio: number;
  linkProps: (surface: ConcurrencyAlertSurface) => { to: string };
};

export function useConcurrencyAlert(): ConcurrencyAlert | null {
  const [{ data, error, fetching }] = useQuery({
    query: ConcurrencyAlertCheckDocument,
    requestPolicy: 'cache-first',
  });

  const [now, setNow] = useState<number | null>(null);

  useEffect(() => {
    if (!data) {
      return;
    }
    const at = receivedAt.get(data) ?? Date.now();
    receivedAt.set(data, at);
    setNow(at);
  }, [data]);

  const burst = data?.account.burstConcurrency;
  const boosted =
    burst != null &&
    now != null &&
    isBurstActive({
      bursts: burst.usage.bursts,
      exhaustedAt: burst.exhaustedAt,
      now,
    });

  const burstSettled = now != null || (error != null && !fetching);

  const { isPressured, isMarketplace, isFreePlan } =
    useAccountConcurrencyPressure({ skip: !burstSettled || boosted });

  const state = alertState({ boosted, pressured: isPressured });
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
        isMarketplace,
        isFreePlan,
        marketplaceBillingURL: data?.account.marketplaceBillingURL,
      }),
  };
}
