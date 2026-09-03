import { useCallback, useEffect, useRef, useState } from 'react';
import { Banner } from '@inngest/components/Banner';
import { Button } from '@inngest/components/Button';
import { ulid } from 'ulid';

import { useEnvironment } from '@/components/Environments/environment-context';
import {
  trackBannerCTAClicked,
  trackBannerDismissed,
  trackBannerViewed,
} from '@/utils/analyticsEvents';
import { pathCreator } from '@/utils/urls';
import {
  isDismissalActive,
  nextImpression,
  readDismissedAt,
  resolveBillingCTA,
  writeDismissedAt,
  type BannerImpression,
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

  // Null for marketplace accounts, and until the plan resolves. Settled by the
  // time the banner is visible: the plan rides on the same identity query as
  // the account id, and nothing renders before that resolves.
  const billingCTA = resolveBillingCTA({ isFreePlan, isMarketplace });

  // The current appearance of the banner. Every event this appearance produces
  // is stamped with its id and reports its snapshot, so clicks and dismissals
  // divide by the impressions they actually came from. Held in state rather
  // than derived, because minting an id is not something render may do twice.
  const [impression, setImpression] = useState<BannerImpression | null>(null);

  useEffect(() => {
    setImpression((prev) =>
      nextImpression(
        prev,
        {
          isVisible,
          accountID,
          scope,
          minutesWithHits,
          cta: USAGE_CTA,
          billingCTA,
        },
        ulid,
      ),
    );
  }, [isVisible, accountID, scope, minutesWithHits, billingCTA]);

  // The impression event the click and dismiss rates are measured against.
  // Fires on every appearance — deduping it while leaving the actions undeduped
  // is what made the old rates meaningless. The ref bounds it to once per id
  // regardless of how often the effect re-runs (StrictMode double-invokes it in
  // dev), without putting storage on the denominator's path: a browser that
  // refuses storage must still be able to emit a view, or its clicks land with
  // no denominator at all.
  const trackedImpressionID = useRef<string | null>(null);

  useEffect(() => {
    if (!impression || trackedImpressionID.current === impression.id) {
      return;
    }
    trackedImpressionID.current = impression.id;
    trackBannerViewed({
      feature: ANALYTICS_FEATURE,
      bannerId: BANNER_ID,
      accountId: impression.accountID,
      impressionId: impression.id,
      scope: impression.scope,
      minutesWithHits: impression.minutesWithHits,
      windowMinutes: impression.windowMinutes,
      cta: impression.cta,
      secondaryCta: impression.billingCTA ?? undefined,
    });
  }, [impression]);

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
    // Hiding the banner runs off React state below, not off this write, so a
    // storage failure still hides it for the rest of the session.
    writeDismissedAt(accountID, now);
    setDismissedAt(now);
    trackBannerDismissed({
      feature: ANALYTICS_FEATURE,
      bannerId: BANNER_ID,
      accountId: impression?.accountID ?? accountID,
      impressionId: impression?.id,
      scope: impression?.scope ?? scope,
      minutesWithHits: impression?.minutesWithHits,
      windowMinutes: impression?.windowMinutes,
    });
  }, [accountID, scope, impression]);

  // Both CTAs route client-side, so the page isn't torn down mid-flight and
  // Segment's fire-and-forget track call still lands. Sending either of them
  // through a plain `href` would unload the document and cancel the request,
  // silently deflating that one CTA's click count against the other's.
  const onCTAClick = useCallback(
    (cta: string) => () => {
      trackBannerCTAClicked({
        feature: ANALYTICS_FEATURE,
        bannerId: BANNER_ID,
        accountId: impression?.accountID ?? accountID,
        impressionId: impression?.id,
        scope: impression?.scope ?? scope,
        minutesWithHits: impression?.minutesWithHits,
        windowMinutes: impression?.windowMinutes,
        cta,
      });
    },
    [accountID, scope, impression],
  );

  // Rendering off the impression rather than `isVisible` is what guarantees the
  // CTAs a viewer saw are the CTAs the view event reported.
  if (!impression) {
    return null;
  }

  const { billingCTA: renderedBillingCTA } = impression;

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
          {renderedBillingCTA && (
            <Button
              appearance="outlined"
              kind="secondary"
              size="small"
              label={billingCTACopy[renderedBillingCTA]}
              onClick={onCTAClick(renderedBillingCTA)}
              to={billingCTAPath(renderedBillingCTA)}
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
 *
 * Passed to the button's `to`, not its `href`: `href` renders a bare anchor and
 * full-page navigation would cancel the in-flight click event. As with the
 * usage CTA, the ref rides in the path because the router's search schema
 * doesn't declare it.
 */
function billingCTAPath(cta: ConcurrencyBillingCTA): string {
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
