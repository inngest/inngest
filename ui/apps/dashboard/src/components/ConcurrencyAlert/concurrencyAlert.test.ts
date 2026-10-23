import { describe, expect, it } from 'vitest';

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
  it('sends boosted accounts to the usage burst section', () => {
    expect(concurrencyLinkProps({ state: 'boosted', surface: 'pill' })).toEqual(
      {
        to: '/billing/usage?ref=app-concurrency-boost-pill#burst-concurrency',
      },
    );
  });

  it('sends limited accounts to the usage concurrency section', () => {
    expect(concurrencyLinkProps({ state: 'limited', surface: 'card' })).toEqual(
      {
        to: '/billing/usage?ref=app-concurrency-limited-card#account-concurrency',
      },
    );
  });
});
