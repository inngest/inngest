import { useCallback, useEffect, useState } from 'react';
import { Banner } from '@inngest/components/Banner';
import { Button } from '@inngest/components/Button';

import { useEnvironment } from '@/components/Environments/environment-context';
import { pathCreator } from '@/utils/urls';
import {
  CONCURRENCY_WINDOW_MINUTES,
  isDismissalActive,
  readDismissedAt,
  writeDismissedAt,
} from './accountConcurrency';
import { useAccountConcurrencyPressure } from './useAccountConcurrencyPressure';

/**
 * Warns on the runs list when the account has repeatedly hit its concurrency
 * limit. Renders nothing when there's no pressure or the user dismissed it
 * within the last day.
 */
type Props = {
  // Bumped by the runs page's Refresh action. Since the pressure check doesn't
  // poll, this is what re-evaluates the banner without a full page load.
  refreshNonce?: number;
};

export function AccountConcurrencyBanner({ refreshNonce }: Props) {
  const env = useEnvironment();

  const [dismissedAt, setDismissedAt] = useState<number | null>(null);

  // The function-scoped runs route renders without a ClientOnly wrapper, so
  // this component is server-rendered. Read localStorage in an effect rather
  // than a lazy initializer, matching useBooleanLocalStorage, and hold the
  // banner back until hydration so a previously-dismissed banner never flashes.
  const [isReady, setIsReady] = useState(false);

  // Derived from state set by the effect below, so it's available before the
  // hook call that needs it as `skip`. On the first render nothing is ready
  // yet, which is exactly when we want the expensive query suppressed anyway.
  const isDismissed = isDismissalActive(dismissedAt, Date.now());

  // Don't fetch an answer we can't act on: nothing before we know the dismissal
  // state, and nothing for the 24h the banner stays dismissed.
  const { isPressured, accountID } = useAccountConcurrencyPressure({
    refreshNonce,
    skip: !isReady || isDismissed,
  });

  // Re-read when the account changes: dismissal is stored per account, so
  // switching orgs must consult that account's entry, not the previous one's.
  useEffect(() => {
    if (!accountID) {
      return;
    }
    setDismissedAt(readDismissedAt(accountID));
    setIsReady(true);
  }, [accountID]);

  const onDismiss = useCallback(() => {
    if (!accountID) {
      return;
    }
    const now = Date.now();
    writeDismissedAt(accountID, now);
    setDismissedAt(now);
  }, [accountID]);

  if (!isReady || !isPressured || isDismissed) {
    return null;
  }

  return (
    <Banner
      severity="warning"
      onDismiss={onDismiss}
      cta={
        <Button
          appearance="outlined"
          kind="secondary"
          size="small"
          label="View account concurrency"
          href={pathCreator.metrics({
            envSlug: env.slug,
            ref: 'app-runs-concurrency-banner',
          })}
        />
      }
    >
      Your account reached its concurrency limit repeatedly in the last{' '}
      {CONCURRENCY_WINDOW_MINUTES} minutes, delaying runs.
    </Banner>
  );
}
