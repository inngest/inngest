import { useEffect, useRef } from 'react';
import { ulid } from 'ulid';
import { gql, useQuery, type TypedDocumentNode } from 'urql';

import {
  trackBannerCTAClicked,
  trackBannerViewed,
} from '@/utils/analyticsEvents';

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
  linkProps: { to: string; onClick: () => void };
};

export function useConcurrencyAlert(
  surface: ConcurrencyAlertSurface,
): ConcurrencyAlert | null {
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
  const accountId = data?.account.id;
  const impression = useRef<Parameters<typeof trackBannerCTAClicked>[0] | null>(
    null,
  );

  useEffect(() => {
    if (!state || !accountId) {
      impression.current = null;
      return;
    }

    // Usage refreshes and Strict Mode effect replay are the same appearance.
    if (
      impression.current?.accountId === accountId &&
      impression.current.alertState === state &&
      impression.current.surface === surface
    ) {
      return;
    }

    impression.current = {
      feature: 'account-concurrency',
      bannerId: 'concurrency-alert',
      accountId,
      impressionId: ulid(),
      alertState: state,
      surface,
      cta: state === 'boosted' ? 'view-usage' : 'add-concurrency',
    };
    trackBannerViewed(impression.current);
  }, [accountId, state, surface]);

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
    linkProps: {
      ...concurrencyLinkProps({ state, surface }),
      onClick: () => {
        if (impression.current) {
          trackBannerCTAClicked(impression.current);
        }
      },
    },
  };
}
