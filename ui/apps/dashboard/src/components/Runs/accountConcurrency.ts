/**
 * Pure helpers behind the account-concurrency banner on the runs list. Kept
 * apart from the hook so the threshold and dismissal rules are testable
 * without a GraphQL client or a fake clock in React.
 */

// How far back we look for account-level concurrency limit hits. The metrics
// resolver derives bucket size from the range, and anything under 3h yields
// 1-minute buckets, so this window is 10 buckets wide.
export const CONCURRENCY_WINDOW_MS = 10 * 60 * 1000;

// The banner copy names this window, so derive it rather than hardcoding the
// number in the string — otherwise tuning the window silently makes the copy lie.
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
 * True when at least CONCURRENCY_HIT_MINUTES_THRESHOLD of the returned buckets
 * recorded a limit hit. Counts buckets rather than summing values so a single
 * minute of heavy contention can't trip the banner on its own.
 */
export function isConcurrencyPressured(buckets: Bucket[] | undefined): boolean {
  if (!buckets) return false;

  const minutesWithHits = buckets.filter(({ value }) => value > 0).length;
  return minutesWithHits >= CONCURRENCY_HIT_MINUTES_THRESHOLD;
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
