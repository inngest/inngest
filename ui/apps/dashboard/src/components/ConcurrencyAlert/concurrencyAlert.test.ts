import { describe, expect, it } from 'vitest';

import { pathCreator } from '@/utils/urls';
import {
  alertState,
  concurrencyLinkProps,
  effectiveLimit,
  usedRatio,
} from './concurrencyAlert';

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
