/**
 * Pure helpers behind the account-concurrency banner on the runs list. Kept
 * apart from the hook so the threshold and dismissal rules are testable
 * without a GraphQL client or a fake clock in React.
 */

// How far back we look for account-level concurrency limit hits. The metrics
// resolver derives bucket size from the range, and anything under 3h yields
// 1-minute buckets, so this window is 10 buckets wide.
export const CONCURRENCY_WINDOW_MS = 10 * 60 * 1000;

// Reported on the banner's view event so reactions can be read against the
// window the signal was measured over. Derived rather than hardcoded so tuning
// the window can't silently desync the two.
export const CONCURRENCY_WINDOW_MINUTES = CONCURRENCY_WINDOW_MS / 60_000;

// How far behind "now" the window ends. The newest bucket is the least
// trustworthy — a minute still in progress reads the same as a quiet one — so
// we drop it rather than let it count as a zero. One minute is a deliberately
// conservative buffer, not a measured figure.
export const CONCURRENCY_END_OFFSET_MS = 60 * 1000;

// Number of distinct minutes within the window that must have recorded at
// least one limit hit before we claim runs are being delayed. Above 1 so a
// single blip of contention doesn't raise the banner.
export const CONCURRENCY_HIT_MINUTES_THRESHOLD = 3;

// Metric namespace incremented whenever the queue refuses to admit an item
// into processing because the account concurrency limit was reached. Edge-
// triggered on a blocked attempt, not level-triggered on being at capacity: an
// account sitting at its limit with no new work arriving increments nothing.
// So every increment is a run that was actually held back.
export const ACCOUNT_CONCURRENCY_LIMIT_METRIC =
  'account_concurrency_limit_reached_total';

const DISMISSAL_STORAGE_KEY_PREFIX = 'dismissedAccountConcurrencyBanner';

/**
 * Dismissal is stored per account. The condition is account-wide, but
 * localStorage is per browser origin — an unkeyed entry would let a dismissal
 * on one account silently suppress the banner on another the user switches to.
 */
export function dismissalStorageKey(accountID: string): string {
  return `${DISMISSAL_STORAGE_KEY_PREFIX}:${accountID}`;
}

// Re-arm the banner a day after dismissal: an account that fixes its
// concurrency and hits the limit again next month should hear about it.
export const DISMISSAL_LIFETIME_MS = 24 * 60 * 60 * 1000;

export function floorToMinute(ms: number): number {
  return Math.floor(ms / 60_000) * 60_000;
}

export type ConcurrencyWindow = {
  from: string;
  to: string;
};

/**
 * The time range to request limit-hit buckets for: CONCURRENCY_WINDOW_MS wide,
 * ending CONCURRENCY_END_OFFSET_MS before now so the most recent, possibly
 * incomplete minute stays out of the sample.
 *
 * Both bounds are floored to the minute to match the bucket granularity. That
 * also makes repeated calls within the same minute return identical bounds, so
 * a rapid Refresh hits the urql document cache instead of re-querying.
 */
export function buildConcurrencyWindow(now: number): ConcurrencyWindow {
  const end = floorToMinute(now) - CONCURRENCY_END_OFFSET_MS;

  return {
    from: new Date(end - CONCURRENCY_WINDOW_MS).toISOString(),
    to: new Date(end).toISOString(),
  };
}

type Bucket = { value: number };

/**
 * How many of the returned minute buckets recorded at least one limit hit.
 * Counting buckets rather than summing values is what keeps a single minute of
 * heavy contention from standing in for sustained pressure. Also reported on
 * the banner's view event, so reactions can be read against signal strength.
 */
export function countMinutesWithHits(buckets: Bucket[] | undefined): number {
  if (!buckets) return 0;

  return buckets.filter(({ value }) => value > 0).length;
}

/**
 * True when at least CONCURRENCY_HIT_MINUTES_THRESHOLD of the returned buckets
 * recorded a limit hit.
 */
export function isConcurrencyPressured(buckets: Bucket[] | undefined): boolean {
  return countMinutesWithHits(buckets) >= CONCURRENCY_HIT_MINUTES_THRESHOLD;
}

/**
 * Which billing CTA the banner offers alongside "View usage", or null for no
 * second CTA at all.
 *
 * Marketplace accounts (Vercel, AWS, DigitalOcean, partner) are billed by the
 * provider, and every billing route but /billing/usage redirects them away
 * (see MarketplaceAccessControl) — so any in-app upgrade CTA is a dead end.
 * We have no installation-level URL to send them to instead, so they get none.
 *
 * Otherwise the label tracks what the user is actually buying: a free account
 * changes tier ("Upgrade"), while a paid one is topping up a single
 * entitlement, not moving tiers ("Increase concurrency").
 */
export type ConcurrencyBillingCTA = 'increase-concurrency' | 'upgrade';

export function resolveBillingCTA({
  isFreePlan,
  isMarketplace,
}: {
  // Undefined until the account query resolves. Withholding the CTA until then
  // is what keeps the wrong label from flashing on a paid account.
  isFreePlan: boolean | undefined;
  isMarketplace: boolean;
}): ConcurrencyBillingCTA | null {
  if (isMarketplace || isFreePlan === undefined) return null;

  return isFreePlan ? 'upgrade' : 'increase-concurrency';
}

/**
 * One appearance of the banner, minted when it becomes visible and carried on
 * every event that appearance produces.
 *
 * The `id` is what lets a click or a dismissal be attributed to the exact
 * impression it came from, so click and dismiss rates share a denominator
 * counted in the same unit they are. Everything else is a snapshot taken at
 * mint time rather than read live at click time: a Refresh can refetch while
 * the banner sits on screen, and without the snapshot a click would report a
 * different signal strength than the view it belongs to.
 *
 * The snapshot is also what the banner renders from, so the CTAs a viewer saw
 * and the CTAs reported on the view event cannot disagree.
 */
export type BannerImpression = {
  id: string;
  // Held to detect an org switch as a new appearance, and reported on every
  // event: the warehouse's event rows carry only a user id, and a user in two
  // accounts can't be attributed from membership alone.
  accountID: string;
  minutesWithHits: number;
  windowMinutes: number;
  scope: string | undefined;
  cta: string;
  billingCTA: ConcurrencyBillingCTA | null;
};

type ImpressionInput = {
  isVisible: boolean;
  accountID: string | undefined;
  scope: string | undefined;
  minutesWithHits: number;
  cta: string;
  billingCTA: ConcurrencyBillingCTA | null;
};

/**
 * The impression that should be current, given the one that already is.
 *
 * Returns `prev` by identity whenever the banner is still the same appearance,
 * so the caller's tracking effect stays a no-op. A new impression is minted
 * only on a genuine new appearance: the banner becoming visible, or the
 * account or scope changing underneath it. Notably `minutesWithHits` changing
 * does NOT mint one — the pressure reading moved, the appearance did not.
 *
 * `mintID` is injected so tests get deterministic ids without mocking ulid.
 */
export function nextImpression(
  prev: BannerImpression | null,
  input: ImpressionInput,
  mintID: () => string,
): BannerImpression | null {
  if (!input.isVisible || !input.accountID) return null;

  if (
    prev &&
    prev.accountID === input.accountID &&
    prev.scope === input.scope
  ) {
    return prev;
  }

  return {
    id: mintID(),
    accountID: input.accountID,
    minutesWithHits: input.minutesWithHits,
    windowMinutes: CONCURRENCY_WINDOW_MINUTES,
    scope: input.scope,
    cta: input.cta,
    billingCTA: input.billingCTA,
  };
}

/**
 * Whether a dismissal recorded at `dismissedAt` still suppresses the banner.
 * A malformed or future-dated stamp is treated as expired so a bad write can't
 * hide the banner indefinitely.
 */
export function isDismissalActive(
  dismissedAt: number | null,
  now: number,
): boolean {
  if (dismissedAt === null || !Number.isFinite(dismissedAt)) return false;
  if (dismissedAt > now) return false;

  return now - dismissedAt < DISMISSAL_LIFETIME_MS;
}

export function readDismissedAt(accountID: string): number | null {
  const key = dismissalStorageKey(accountID);

  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return null;

    const parsed = Number(raw);
    return Number.isFinite(parsed) ? parsed : null;
  } catch (error) {
    console.warn(`error reading localStorage key "${key}":`, error);
    return null;
  }
}

export function writeDismissedAt(accountID: string, now: number): void {
  const key = dismissalStorageKey(accountID);

  try {
    window.localStorage.setItem(key, String(now));
  } catch (error) {
    console.warn(`error writing localStorage key "${key}":`, error);
  }
}
