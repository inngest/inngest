import { describe, expect, it } from 'vitest';

import { pathCreator } from '@/utils/urls';
import {
  alertState,
  concurrencyLinkProps,
  effectiveLimit,
  isBurstActive,
  usedRatio,
} from './concurrencyAlert';

// A real minute boundary: 2023-11-14T22:14:00.000Z.
const minuteStart = 1_700_000_040_000;
const now = minuteStart + 37_500;
const iso = (ms: number) => new Date(ms).toISOString();
const minutes = (n: number) => n * 60_000;

describe('isBurstActive', () => {
  it('is boosted while the latest burst is ongoing', () => {
    const bursts = [
      { endedAt: iso(minuteStart - minutes(30)) },
      { endedAt: iso(minuteStart) },
    ];
    expect(isBurstActive({ bursts, exhaustedAt: null, now })).toBe(true);
  });

  it('is boosted when the open minute is counted', () => {
    const bursts = [{ endedAt: iso(minuteStart + minutes(1)) }];
    expect(isBurstActive({ bursts, exhaustedAt: null, now })).toBe(true);
  });

  it('is boosted at the edge of the lag window', () => {
    const bursts = [{ endedAt: iso(minuteStart - minutes(2)) }];
    expect(isBurstActive({ bursts, exhaustedAt: null, now })).toBe(true);
  });

  it('is not boosted once the burst ended beyond the lag window', () => {
    const bursts = [{ endedAt: iso(minuteStart - minutes(3)) }];
    expect(isBurstActive({ bursts, exhaustedAt: null, now })).toBe(false);
  });

  it('is not boosted once the budget is exhausted', () => {
    const bursts = [{ endedAt: iso(minuteStart) }];
    expect(isBurstActive({ bursts, exhaustedAt: iso(minuteStart), now })).toBe(
      false,
    );
  });

  it('is not boosted without bursts', () => {
    expect(isBurstActive({ bursts: [], exhaustedAt: null, now })).toBe(false);
  });
});

describe('alertState', () => {
  it('is limited when only pressured', () => {
    expect(alertState({ boosted: false, pressured: true })).toBe('limited');
  });

  it('prefers boosted over limited', () => {
    expect(alertState({ boosted: true, pressured: true })).toBe('boosted');
  });

  it('is null when neither', () => {
    expect(alertState({ boosted: false, pressured: false })).toBeNull();
  });
});

describe('effectiveLimit', () => {
  it('uses the burst limit while boosted', () => {
    expect(
      effectiveLimit({ state: 'boosted', burstLimit: 300, baseLimit: 100 }),
    ).toBe(300);
  });

  it('uses the base limit while limited', () => {
    expect(
      effectiveLimit({ state: 'limited', burstLimit: 300, baseLimit: 100 }),
    ).toBe(100);
  });
});

describe('usedRatio', () => {
  it('reports the used share of the budget', () => {
    expect(usedRatio({ budgetMinutes: 100, minutesRemaining: 25 })).toBe(0.75);
  });

  it('clamps to 0 and 1', () => {
    expect(usedRatio({ budgetMinutes: 100, minutesRemaining: 150 })).toBe(0);
    expect(usedRatio({ budgetMinutes: 100, minutesRemaining: -10 })).toBe(1);
  });
});

describe('concurrencyLinkProps', () => {
  const base = {
    surface: 'pill' as const,
    isMarketplace: false,
    isFreePlan: false,
    marketplaceBillingURL: null,
  };

  it('sends boosted accounts to usage', () => {
    expect(concurrencyLinkProps({ ...base, state: 'boosted' })).toEqual({
      to: pathCreator.billingUsage({ ref: 'app-concurrency-boost-pill' }),
    });
  });

  it('sends limited marketplace accounts to their marketplace billing', () => {
    expect(
      concurrencyLinkProps({
        ...base,
        state: 'limited',
        isMarketplace: true,
        marketplaceBillingURL: 'https://vercel.com/billing',
      }),
    ).toEqual({ to: 'https://vercel.com/billing' });
  });

  it('sends limited free accounts to plans', () => {
    expect(
      concurrencyLinkProps({ ...base, state: 'limited', isFreePlan: true }),
    ).toEqual({
      to: pathCreator.billing({
        tab: 'plans',
        ref: 'app-concurrency-limited-pill',
      }),
    });
  });

  it('sends limited paid accounts to the concurrency addon', () => {
    expect(
      concurrencyLinkProps({ ...base, state: 'limited', surface: 'card' }),
    ).toEqual({
      to: pathCreator.billing({
        highlight: 'concurrency',
        ref: 'app-concurrency-limited-card',
      }),
    });
  });
});
