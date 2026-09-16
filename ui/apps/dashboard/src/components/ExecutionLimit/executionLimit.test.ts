import { describe, expect, it } from 'vitest';

import {
  dismissalStorageKey,
  DISMISSAL_LIFETIME_MS,
  isDismissalActive,
  isExecutionCapped,
  legacyExecutionCap,
  usageBand,
} from './executionLimit';

const atLimit = {
  usage: 50_000,
  limit: 50_000,
  enforced: true,
  overageAllowed: false,
};

describe('isExecutionCapped', () => {
  it('caps once usage reaches the limit', () => {
    expect(isExecutionCapped(atLimit)).toBe(true);
    expect(isExecutionCapped({ ...atLimit, usage: 49_999 })).toBe(false);
  });

  it('never caps while enforcement is off', () => {
    expect(isExecutionCapped({ ...atLimit, enforced: false })).toBe(false);
  });

  it('never caps accounts billed for overage', () => {
    expect(isExecutionCapped({ ...atLimit, overageAllowed: true })).toBe(false);
  });
});

describe('legacyExecutionCap', () => {
  const legacyAtLimit = { usage: 50_000, limit: 50_000, overageAllowed: false };
  const cappedBy = (executions: typeof legacyAtLimit) => {
    const cap = legacyExecutionCap(executions);
    return cap && isExecutionCapped(cap);
  };

  it('treats unlimited plans as having no cap', () => {
    expect(legacyExecutionCap({ ...legacyAtLimit, limit: null })).toBeNull();
  });

  it('caps at the limit as if enforced', () => {
    expect(cappedBy(legacyAtLimit)).toBe(true);
    expect(cappedBy({ ...legacyAtLimit, usage: 49_999 })).toBe(false);
  });

  it('never caps accounts billed for overage', () => {
    expect(cappedBy({ ...legacyAtLimit, overageAllowed: true })).toBe(false);
  });
});

describe('usageBand', () => {
  const band = (usage: number, isCapped = false) =>
    usageBand({ usage, limit: 50_000, isCapped });

  it('stays quiet below half the limit', () => {
    expect(band(0)).toBe('under50');
    expect(band(24_999)).toBe('under50');
  });

  it('steps up at each threshold', () => {
    expect(band(25_000)).toBe('50');
    expect(band(37_499)).toBe('50');
    expect(band(37_500)).toBe('75');
    expect(band(44_999)).toBe('75');
    expect(band(45_000)).toBe('90');
  });

  it('reports 90 rather than capped at the limit while enforcement is off', () => {
    expect(band(50_000)).toBe('90');
    expect(band(60_000)).toBe('90');
  });

  it('reports capped whenever the cap is enforced, whatever the ratio', () => {
    expect(band(50_000, true)).toBe('capped');
    expect(band(47_500, true)).toBe('capped');
  });

  it('treats a non-positive limit as no usage', () => {
    expect(usageBand({ usage: 10, limit: 0, isCapped: false })).toBe('under50');
  });
});

describe('dismissalStorageKey', () => {
  const accountA = '5d258962-2c37-4a5d-b875-ebe72792c47f';
  const accountB = 'e8ea18c4-dbb4-4e98-a6a4-8ff8b3801765';

  it('gives different accounts different keys', () => {
    expect(dismissalStorageKey('banner', accountA)).not.toBe(
      dismissalStorageKey('banner', accountB),
    );
  });

  it('gives the banner and the card different keys', () => {
    expect(dismissalStorageKey('banner', accountA)).not.toBe(
      dismissalStorageKey('card', accountA),
    );
  });

  it('is stable for the same surface and account', () => {
    expect(dismissalStorageKey('card', accountA)).toBe(
      dismissalStorageKey('card', accountA),
    );
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

  it('treats a future-dated stamp as expired', () => {
    expect(isDismissalActive(now + 60_000, now)).toBe(false);
  });

  it('treats a malformed stamp as expired', () => {
    expect(isDismissalActive(NaN, now)).toBe(false);
  });
});
