import { useCallback, useEffect, useState } from 'react';
import { Banner } from '@inngest/components/Banner';
import { Button } from '@inngest/components/Button';

import { useEnvironment } from '@/components/Environments/environment-context';
import {
  trackBannerCTAClicked,
  trackBannerDismissed,
  trackBannerViewed,
} from '@/utils/analyticsEvents';
import { pathCreator } from '@/utils/urls';
import {
  CONCURRENCY_WINDOW_MINUTES,
  hasTrackedView,
  isDismissalActive,
  markViewTracked,
  readDismissedAt,
  resolveBillingCTA,
  writeDismissedAt,
  type ConcurrencyBillingCTA,
} from './accountConcurrency';
import { useAccountConcurrencyPressure } from './useAccountConcurrencyPressure';

const ANALYTICS_FEATURE = 'account-concurrency' as const;
const BANNER_ID = 'account-concurrency';

// Identifies the always-present CTA on the click event. The billing CTA
// reports its own kind, which doubles as its analytics name.
const USAGE_CTA = 'view-usage';

const billingCTACopy: Record<ConcurrencyBillingCTA, string> = {
  'increase-concurrency': 'Increase concurrency',
  upgrade: 'Upgrade',
};

/**
 * Warns on the runs list when the account has repeatedly hit its concurrency
 * limit. Renders nothing when there's no pressure or the user dismissed it
 * within the last day.
 */
type Props = {
  // Bumped by the runs page's Refresh action. Since the pressure check doesn't
  // poll, this is what re-evaluates the banner without a full page load.
  refreshNonce?: number;
  // Which runs page this is, reported on every analytics event so reactions can
  // be compared between the environment-wide and per-function lists.
  scope?: string;
};

export function AccountConcurrencyBanner({ refreshNonce, scope }: Props) {
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
  const { isPressured, minutesWithHits, accountID, isFreePlan, isMarketplace } =
    useAccountConcurrencyPressure({
      refreshNonce,
      skip: !isReady || isDismissed,
    });

  const isVisible = isReady && isPressured && !isDismissed;

  // Null for marketplace accounts, and until the plan resolves. Deciding this
  // here rather than in the JSX keeps the rendered CTA and the CTA reported on
  // the view event from ever disagreeing.
  const billingCTA = resolveBillingCTA({ isFreePlan, isMarketplace });

  // The impression event the click and dismiss rates are measured against.
  // Deduped per account and scope per session, since the banner re-resolves on
  // every mount and every Refresh.
  useEffect(() => {
    if (!isVisible || !accountID || hasTrackedView(accountID, scope)) {
      return;
    }
    markViewTracked(accountID, scope);
    trackBannerViewed({
      feature: ANALYTICS_FEATURE,
      bannerId: BANNER_ID,
      scope,
      minutesWithHits,
      windowMinutes: CONCURRENCY_WINDOW_MINUTES,
      cta: USAGE_CTA,
      secondaryCta: billingCTA ?? undefined,
    });
  }, [isVisible, accountID, scope, minutesWithHits, billingCTA]);

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
    trackBannerDismissed({
      feature: ANALYTICS_FEATURE,
      bannerId: BANNER_ID,
      scope,
    });
  }, [accountID, scope]);

  // Both CTAs navigate, so this fires during the click that leaves the page.
  // Segment's track call is fire-and-forget, which is why it isn't awaited.
  const onCTAClick = useCallback(
    (cta: string) => () => {
      trackBannerCTAClicked({
        feature: ANALYTICS_FEATURE,
        bannerId: BANNER_ID,
        scope,
        cta,
      });
    },
    [scope],
  );

  if (!isVisible) {
    return null;
  }

  return (
    <Banner
      severity="warning"
      onDismiss={onDismiss}
      cta={
        <div className="flex shrink-0 items-center gap-3">
          <Banner.Link
            severity="warning"
            onClick={onCTAClick(USAGE_CTA)}
            // The ref rides in `to` rather than `search`: the router's search
            // schema doesn't declare `ref`, and this matches how the rest of
            // the app links with one (see OnboardingWidget).
            to={pathCreator.metrics({
              envSlug: env.slug,
              ref: 'app-runs-concurrency-banner-usage',
            })}
          >
            View usage
          </Banner.Link>
          {billingCTA && (
            <Button
              appearance="outlined"
              kind="secondary"
              size="small"
              label={billingCTACopy[billingCTA]}
              onClick={onCTAClick(billingCTA)}
              href={billingCTAHref(billingCTA)}
            />
          )}
        </div>
      }
    >
      Runs delayed because your concurrency limit was reached.
    </Banner>
  );
}

/**
 * Free accounts need a tier change, so they land on the plan picker. Paid
 * accounts are buying more of one entitlement, so they land on the billing
 * overview with the concurrency add-on highlighted — the same destination the
 * metrics dashboard's concurrency CTA uses.
 */
function billingCTAHref(cta: ConcurrencyBillingCTA): string {
  if (cta === 'upgrade') {
    return pathCreator.billing({
      tab: 'plans',
      ref: 'app-runs-concurrency-banner-upgrade',
    });
  }

  return pathCreator.billing({
    highlight: 'concurrency',
    ref: 'app-runs-concurrency-banner-increase-concurrency',
  });
}
