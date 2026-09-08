import { describe, expect, it } from 'vitest';

import {
  buildConcurrencyWindow,
  CONCURRENCY_END_OFFSET_MS,
  CONCURRENCY_WINDOW_MINUTES,
  countMinutesWithHits,
  dismissalStorageKey,
  DISMISSAL_LIFETIME_MS,
  isConcurrencyPressured,
  isDismissalActive,
  nextImpression,
  resolveBillingCTA,
} from './accountConcurrency';

const bucket = (value: number) => ({ value });

describe('buildConcurrencyWindow', () => {
  // A real minute boundary: 2023-11-14T22:14:00.000Z.
  const minuteStart = 1_700_000_040_000;
  const ms = (iso: string) => new Date(iso).getTime();

  it('ends a full offset before the current minute', () => {
    const { to } = buildConcurrencyWindow(minuteStart + 37_500);
    expect(ms(to)).toBe(minuteStart - CONCURRENCY_END_OFFSET_MS);
  });

  it('spans exactly ten one-minute buckets', () => {
    const { from, to } = buildConcurrencyWindow(minuteStart + 37_500);
    expect((ms(to) - ms(from)) / 60_000).toBe(10);
  });

  it('excludes the most recent minute', () => {
    const now = minuteStart + 37_500;
    const { to } = buildConcurrencyWindow(now);
    expect(ms(to)).toBeLessThanOrEqual(now - CONCURRENCY_END_OFFSET_MS);
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

describe('resolveBillingCTA', () => {
  it('offers a tier change on a free plan', () => {
    expect(resolveBillingCTA({ isFreePlan: true, isMarketplace: false })).toBe(
      'upgrade',
    );
  });

  it('offers an entitlement top-up on a paid plan', () => {
    expect(resolveBillingCTA({ isFreePlan: false, isMarketplace: false })).toBe(
      'increase-concurrency',
    );
  });

  // Every billing route but /billing/usage redirects marketplace accounts
  // away, so a billing CTA could only dead-end them.
  it('offers nothing to a marketplace account, whatever its plan', () => {
    expect(resolveBillingCTA({ isFreePlan: true, isMarketplace: true })).toBe(
      null,
    );
    expect(resolveBillingCTA({ isFreePlan: false, isMarketplace: true })).toBe(
      null,
    );
  });

  // Withheld rather than guessed: a default would flash the wrong label on
  // whichever plan it guessed against.
  it('offers nothing until the plan is known', () => {
    expect(
      resolveBillingCTA({ isFreePlan: undefined, isMarketplace: false }),
    ).toBe(null);
  });
});

describe('nextImpression', () => {
  const visible = {
    isVisible: true,
    accountID: 'acct-1',
    scope: 'env',
    minutesWithHits: 4,
    cta: 'view-usage',
    billingCTA: 'upgrade' as const,
  };

  // Deterministic ids, so a test can assert "a new one" without matching ULIDs.
  function counter() {
    let n = 0;
    return () => `imp-${++n}`;
  }

  it('has no impression while the banner is hidden', () => {
    expect(
      nextImpression(null, { ...visible, isVisible: false }, counter()),
    ).toBe(null);
  });

  // The dismissal key and every event need it, so an impression without one
  // would be unattributable.
  it('has no impression before the account resolves', () => {
    expect(
      nextImpression(null, { ...visible, accountID: undefined }, counter()),
    ).toBe(null);
  });

  it('mints one when the banner becomes visible', () => {
    const impression = nextImpression(null, visible, counter());

    expect(impression).toMatchObject({
      id: 'imp-1',
      accountID: 'acct-1',
      scope: 'env',
      minutesWithHits: 4,
      cta: 'view-usage',
      billingCTA: 'upgrade',
    });
  });

  it('records the window the signal was measured over', () => {
    expect(nextImpression(null, visible, counter())?.windowMinutes).toBe(
      CONCURRENCY_WINDOW_MINUTES,
    );
  });

  // Identity, not just equality: the caller's tracking effect keys on the
  // impression object, so a fresh object would re-fire the view event.
  it('keeps the same impression across re-renders', () => {
    const first = nextImpression(null, visible, counter());

    expect(nextImpression(first, visible, counter())).toBe(first);
  });

  // A Refresh can refetch under a banner that never left the screen. That's a
  // new reading, not a new appearance, and the snapshot stays frozen so a later
  // click reports the same numbers as the view it belongs to.
  it('does not re-mint when the pressure reading changes underneath it', () => {
    const first = nextImpression(null, visible, counter());
    const next = nextImpression(
      first,
      { ...visible, minutesWithHits: 9 },
      counter(),
    );

    expect(next).toBe(first);
    expect(next?.minutesWithHits).toBe(4);
  });

  // The case the old session-scoped dedup silently dropped: pressure clears and
  // returns, or a dismissal lapses, without the tab ever reloading.
  it('mints a new impression when the banner reappears', () => {
    const mint = counter();
    const first = nextImpression(null, visible, mint);
    const hidden = nextImpression(
      first,
      { ...visible, isVisible: false },
      mint,
    );
    const second = nextImpression(hidden, visible, mint);

    expect(hidden).toBe(null);
    expect(second?.id).toBe('imp-2');
  });

  it('mints a new impression when the scope changes', () => {
    const mint = counter();
    const first = nextImpression(null, visible, mint);
    const second = nextImpression(first, { ...visible, scope: 'fn' }, mint);

    expect(second?.id).toBe('imp-2');
    expect(second?.scope).toBe('fn');
  });

  it('mints a new impression when the account changes', () => {
    const mint = counter();
    const first = nextImpression(null, visible, mint);
    const second = nextImpression(
      first,
      { ...visible, accountID: 'acct-2' },
      mint,
    );

    expect(second?.id).toBe('imp-2');
  });
});
