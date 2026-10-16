import { describe, expect, it } from 'vitest';

import {
  CONCURRENCY_WINDOW_MINUTES,
  dismissalStorageKey,
  DISMISSAL_LIFETIME_MS,
  isDismissalActive,
  nextImpression,
  resolveBillingCTA,
} from './accountConcurrency';

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
