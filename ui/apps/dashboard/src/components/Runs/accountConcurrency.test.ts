import { describe, expect, it } from 'vitest';

import {
  buildConcurrencyWindow,
  CONCURRENCY_LAG_OFFSET_MS,
  countMinutesWithHits,
  dismissalStorageKey,
  DISMISSAL_LIFETIME_MS,
  isConcurrencyPressured,
  isDismissalActive,
  viewTrackedStorageKey,
} from './accountConcurrency';

const bucket = (value: number) => ({ value });

describe('buildConcurrencyWindow', () => {
  // A real minute boundary: 2023-11-14T22:14:00.000Z.
  const minuteStart = 1_700_000_040_000;
  const ms = (iso: string) => new Date(iso).getTime();

  it('ends a full lag offset before the current minute', () => {
    const { to } = buildConcurrencyWindow(minuteStart + 37_500);
    expect(ms(to)).toBe(minuteStart - CONCURRENCY_LAG_OFFSET_MS);
  });

  it('spans exactly ten one-minute buckets', () => {
    const { from, to } = buildConcurrencyWindow(minuteStart + 37_500);
    expect((ms(to) - ms(from)) / 60_000).toBe(10);
  });

  it('excludes the most recent minute, which may not be ingested yet', () => {
    const now = minuteStart + 37_500;
    const { to } = buildConcurrencyWindow(now);
    expect(ms(to)).toBeLessThanOrEqual(now - CONCURRENCY_LAG_OFFSET_MS);
  });

  it('floors both bounds to the minute', () => {
    const { from, to } = buildConcurrencyWindow(minuteStart + 37_500);
    expect(ms(from) % 60_000).toBe(0);
    expect(ms(to) % 60_000).toBe(0);
  });

  it('is stable for every instant within the same minute', () => {
    const start = buildConcurrencyWindow(minuteStart);
    expect(buildConcurrencyWindow(minuteStart + 30_000)).toEqual(start);
    expect(buildConcurrencyWindow(minuteStart + 59_999)).toEqual(start);
  });

  it('advances by one minute at the next minute boundary', () => {
    const before = buildConcurrencyWindow(minuteStart + 59_999);
    const after = buildConcurrencyWindow(minuteStart + 60_000);

    expect(ms(after.to) - ms(before.to)).toBe(60_000);
  });

  // Refresh re-resolves the window. Identical bounds within a minute mean the
  // urql document cache absorbs rapid refreshes instead of re-querying.
  it('returns identical bounds for repeated refreshes in the same minute', () => {
    const first = buildConcurrencyWindow(minuteStart + 1_000);
    const second = buildConcurrencyWindow(minuteStart + 45_000);

    expect(second).toEqual(first);
  });
});

describe('isConcurrencyPressured', () => {
  it('is false when no data has loaded yet', () => {
    expect(isConcurrencyPressured(undefined)).toBe(false);
  });

  it('is false for an empty window', () => {
    expect(isConcurrencyPressured([])).toBe(false);
  });

  it('is false when no bucket recorded a hit', () => {
    expect(isConcurrencyPressured([0, 0, 0, 0].map(bucket))).toBe(false);
  });

  it('is false below the threshold, even for a large single spike', () => {
    expect(isConcurrencyPressured([9999, 0, 0].map(bucket))).toBe(false);
    expect(isConcurrencyPressured([1, 1, 0].map(bucket))).toBe(false);
  });

  it('is true once three separate minutes recorded hits', () => {
    expect(isConcurrencyPressured([1, 0, 1, 0, 1].map(bucket))).toBe(true);
  });

  it('counts minutes rather than summing hits', () => {
    // Two minutes totalling 200 hits stays quiet; three minutes totalling 3
    // does not.
    expect(isConcurrencyPressured([100, 100].map(bucket))).toBe(false);
    expect(isConcurrencyPressured([1, 1, 1].map(bucket))).toBe(true);
  });
});

describe('countMinutesWithHits', () => {
  it('is 0 when no data has loaded yet', () => {
    expect(countMinutesWithHits(undefined)).toBe(0);
  });

  // Reported on the view event, so it has to describe breadth of pressure
  // rather than volume — otherwise it can't be compared against the threshold.
  it('counts minutes rather than summing hits', () => {
    expect(countMinutesWithHits([500, 500].map(bucket))).toBe(2);
    expect(countMinutesWithHits([1, 1, 1].map(bucket))).toBe(3);
  });

  it('ignores zero-filled buckets', () => {
    expect(countMinutesWithHits([0, 1, 0, 2, 0].map(bucket))).toBe(2);
  });

  it('agrees with the pressure threshold at the boundary', () => {
    const belowThreshold = [1, 1, 0].map(bucket);
    const atThreshold = [1, 1, 1].map(bucket);

    expect(countMinutesWithHits(belowThreshold)).toBe(2);
    expect(isConcurrencyPressured(belowThreshold)).toBe(false);
    expect(countMinutesWithHits(atThreshold)).toBe(3);
    expect(isConcurrencyPressured(atThreshold)).toBe(true);
  });
});

describe('viewTrackedStorageKey', () => {
  const accountA = '5d258962-2c37-4a5d-b875-ebe72792c47f';
  const accountB = 'e8ea18c4-dbb4-4e98-a6a4-8ff8b3801765';

  it('gives different accounts different keys', () => {
    expect(viewTrackedStorageKey(accountA)).not.toBe(
      viewTrackedStorageKey(accountB),
    );
  });

  // Distinct namespaces: dismissal is durable and per-account, the view marker
  // is session-scoped. Sharing a key would make dismissing suppress impressions.
  it('does not collide with the dismissal key', () => {
    expect(viewTrackedStorageKey(accountA)).not.toBe(
      dismissalStorageKey(accountA),
    );
  });
});

describe('dismissalStorageKey', () => {
  const accountA = '5d258962-2c37-4a5d-b875-ebe72792c47f';
  const accountB = 'e8ea18c4-dbb4-4e98-a6a4-8ff8b3801765';

  // localStorage is per browser origin, so without the account in the key a
  // dismissal on one account would suppress the banner on another.
  it('gives different accounts different keys', () => {
    expect(dismissalStorageKey(accountA)).not.toBe(
      dismissalStorageKey(accountB),
    );
  });

  it('is stable for the same account', () => {
    expect(dismissalStorageKey(accountA)).toBe(dismissalStorageKey(accountA));
  });

  it('includes the account id', () => {
    expect(dismissalStorageKey(accountA)).toContain(accountA);
  });
});

describe('isDismissalActive', () => {
  const now = 1_700_000_000_000;

  it('is false when never dismissed', () => {
    expect(isDismissalActive(null, now)).toBe(false);
  });

  it('is true immediately after dismissal', () => {
    expect(isDismissalActive(now, now)).toBe(true);
  });

  it('is true just inside the lifetime', () => {
    expect(isDismissalActive(now - DISMISSAL_LIFETIME_MS + 1, now)).toBe(true);
  });

  it('expires exactly at the lifetime boundary', () => {
    expect(isDismissalActive(now - DISMISSAL_LIFETIME_MS, now)).toBe(false);
  });

  it('is false long after dismissal', () => {
    expect(isDismissalActive(now - 30 * DISMISSAL_LIFETIME_MS, now)).toBe(
      false,
    );
  });

  it('treats a future-dated stamp as expired', () => {
    expect(isDismissalActive(now + 60_000, now)).toBe(false);
  });

  it('treats a malformed stamp as expired', () => {
    expect(isDismissalActive(NaN, now)).toBe(false);
  });
});
