import { useEffect, useRef } from 'react';
import { useQuery } from 'urql';

import { ConcurrencyAlertCheckDocument } from '@/components/ConcurrencyAlert/useConcurrencyAlert';

type AccountConcurrencyPressure = {
  isPressured: boolean;
  // How many minutes in the window recorded hits. Reported on the view event so
  // reactions can be read against how strong the signal was.
  minutesWithHits: number;
  // Undefined until the query resolves. The banner can't key its
  // dismissal — or render at all — before this is known.
  accountID: string | undefined;
  // Whether the account is billed through a marketplace provider. Assumed
  // false until the query resolves; the banner only uses it to
  // suppress a CTA, and the CTA is withheld while the plan is unknown anyway.
  isMarketplace: boolean;
  // Undefined until the query resolves, or when the account has no
  // plan attached.
  isFreePlan: boolean | undefined;
};

/**
 * Reports whether the account has repeatedly hit its concurrency limit in the
 * recent past, meaning the queue is holding runs back.
 *
 * Deliberately does NOT poll. Status is fetched once per mount, and again
 * whenever `refreshNonce` changes (the runs page's Refresh action). Polling
 * would make cost scale with how long a tab stays open — a background tab left
 * up all day costs hundreds of ClickHouse reads for a banner nobody is looking
 * at — whereas this scales with page loads. The tradeoff is that pressure
 * starting after load won't raise the banner until the next load or refresh.
 *
 * Shares the layout concurrency alert's query so the two can't disagree and
 * urql serves both from one request.
 */
export function useAccountConcurrencyPressure({
  refreshNonce = 0,
}: { refreshNonce?: number } = {}): AccountConcurrencyPressure {
  const [{ data }, reexecute] = useQuery({
    query: ConcurrencyAlertCheckDocument,
  });

  const lastRefreshNonce = useRef(refreshNonce);

  useEffect(() => {
    if (lastRefreshNonce.current === refreshNonce) {
      return;
    }
    lastRefreshNonce.current = refreshNonce;
    reexecute({ requestPolicy: 'network-only' });
  }, [refreshNonce, reexecute]);

  const account = data?.account;

  return {
    accountID: account?.id,
    isFreePlan: account?.plan?.isFree,
    isMarketplace: Boolean(account?.marketplace),
    isPressured: account?.concurrencyStatus?.limitReached ?? false,
    minutesWithHits: account?.concurrencyStatus?.minutesWithHits ?? 0,
  };
}
