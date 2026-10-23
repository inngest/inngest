import { describe, expect, it } from 'vitest';

import {
  dismissalStorageKey,
  DISMISSAL_LIFETIME_MS,
  isDismissalActive,
  isCapHit,
  legacyExecutionCap,
  pillContent,
  usageBand,
} from './executionLimit';

const atLimit = {
  usage: 50_000,
  limit: 50_000,
  overageLimit: 0,
  enforced: true,
  exceeded: true,
};

describe('isCapHit', () => {
  it('hits the cap once the backend reports it exceeded', () => {
    expect(isCapHit(atLimit)).toBe(true);
    expect(isCapHit({ ...atLimit, exceeded: false })).toBe(false);
  });

  it('never hits the cap while enforcement is off', () => {
    expect(isCapHit({ ...atLimit, enforced: false })).toBe(false);
  });
});

describe('legacyExecutionCap', () => {
  const legacyAtLimit = { usage: 50_000, limit: 50_000, overageAllowed: false };
  const capHitBy = (executions: typeof legacyAtLimit) => {
    const cap = legacyExecutionCap(executions);
    return cap && isCapHit(cap);
  };

  it('treats unlimited plans as having no cap', () => {
    expect(legacyExecutionCap({ ...legacyAtLimit, limit: null })).toBeNull();
  });

  it('hits the cap at the limit as if enforced', () => {
    expect(capHitBy(legacyAtLimit)).toBe(true);
    expect(capHitBy({ ...legacyAtLimit, usage: 49_999 })).toBe(false);
  });

  it('treats accounts billed for overage as having no cap', () => {
    expect(
      legacyExecutionCap({ ...legacyAtLimit, overageAllowed: true }),
    ).toBeNull();
  });
});

describe('usageBand', () => {
  const band = (usage: number, capHit = false, overageLimit = 0) =>
    usageBand({ usage, limit: 50_000, overageLimit, capHit });

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

  it('never reports limitReached past the limit unless the cap is hit', () => {
    expect(band(50_000)).toBe('90');
    expect(band(60_000)).toBe('90');
  });

  it('reports a grace period past the limit when extra executions are granted', () => {
    expect(band(49_999, false, 10_000)).toBe('90');
    expect(band(50_000, false, 10_000)).toBe('grace');
    expect(band(60_000, true, 10_000)).toBe('limitReached');
  });

  it('reports limitReached whenever the cap is hit, whatever the ratio', () => {
    expect(band(50_000, true)).toBe('limitReached');
    expect(band(47_500, true)).toBe('limitReached');
  });

  it('treats a non-positive limit as no usage', () => {
    expect(
      usageBand({ usage: 10, limit: 0, overageLimit: 0, capHit: false }),
    ).toBe('under50');
    expect(
      usageBand({ usage: 10, limit: 0, overageLimit: 10_000, capHit: false }),
    ).toBe('under50');
  });
});

describe('dismissalStorageKey', () => {
  const accountA = '5d258962-2c37-4a5d-b875-ebe72792c47f';
  const accountB = 'e8ea18c4-dbb4-4e98-a6a4-8ff8b3801765';

  it('gives different accounts different keys', () => {
    expect(dismissalStorageKey('card', accountA, '50')).not.toBe(
      dismissalStorageKey('card', accountB, '50'),
    );
  });

  it('gives different bands different keys', () => {
    expect(dismissalStorageKey('card', accountA, '50')).not.toBe(
      dismissalStorageKey('card', accountA, '75'),
    );
  });

  it('is stable for the same surface, account and band', () => {
    expect(dismissalStorageKey('card', accountA, '50')).toBe(
      dismissalStorageKey('card', accountA, '50'),
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

describe('pillContent', () => {
  it('stays hidden below half of the limit', () => {
    expect(pillContent('under50', false, 0)).toBeNull();
  });

  it('cautions once usage crosses half of the limit', () => {
    expect(pillContent('50', false, 0)).toMatchObject({ kind: 'caution' });
    expect(pillContent('50', false, 0)?.text).toContain('almost reached');
  });

  it('warns at three-quarters of the limit', () => {
    expect(pillContent('75', false, 0)).toMatchObject({ kind: 'warning' });
    expect(pillContent('75', false, 0)?.text).toContain('almost reached');
  });

  it('errors at 90% of the limit, keeping the almost-reached copy', () => {
    expect(pillContent('90', false, 0)).toMatchObject({ kind: 'error' });
    expect(pillContent('90', false, 0)?.text).toContain('almost reached');
  });

  it('escalates to an error once the limit is reached, dropping the almost-reached copy', () => {
    expect(pillContent('limitReached', false, 0)).toMatchObject({
      kind: 'error',
    });
    expect(pillContent('limitReached', false, 0)?.text).not.toContain('almost');
  });

  it('points Vercel accounts at the integration settings', () => {
    expect(pillContent('75', true, 0)?.text).toContain('Vercel');
  });

  it('warns about the grace period, pointing Vercel accounts at the integration settings', () => {
    expect(pillContent('grace', false, 8_000)).toMatchObject({
      kind: 'warning',
    });
    expect(pillContent('grace', false, 8_000)?.text).toContain(
      '8,000 executions left',
    );
    expect(pillContent('grace', false, 8_000)?.text).not.toContain('Vercel');
    expect(pillContent('grace', true, 8_000)?.text).toContain(
      'Vercel Integrations Settings',
    );
  });
});
