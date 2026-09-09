import * as Sentry from '@sentry/tanstackstart-react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { pickSelfServePlans, type SelfServePlan } from './utils';

vi.mock('@sentry/tanstackstart-react', () => ({
  captureMessage: vi.fn(),
}));

function plan(
  values: Pick<
    SelfServePlan,
    'amount' | 'isFree' | 'isLegacy' | 'name' | 'slug'
  >,
): SelfServePlan {
  return {
    ...values,
    billingPeriod: 'month',
    entitlements: {
      concurrency: { limit: values.isFree ? 5 : 100 },
      eventSize: { limit: 256 },
      executions: { limit: null },
      history: { limit: 7 },
      runCount: { limit: null },
      stepCount: { limit: null },
    },
    id: values.slug,
  };
}

describe('pickSelfServePlans', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('selects current self-serve plans without encoding versioned slugs', () => {
    const plans = [
      plan({
        amount: 0,
        isFree: true,
        isLegacy: false,
        name: 'Hobby',
        slug: 'hobby-from-api',
      }),
      plan({
        amount: 9_900,
        isFree: false,
        isLegacy: false,
        name: 'Pro',
        slug: 'pro-from-api',
      }),
      plan({
        amount: 7_500,
        isFree: false,
        isLegacy: true,
        name: 'Pro',
        slug: 'retired-pro',
      }),
    ];

    expect(pickSelfServePlans(plans)).toEqual({
      hobby: plans[0],
      pro: plans[1],
    });
  });

  it('does not treat other free or paid products as Pro', () => {
    expect(
      pickSelfServePlans([
        plan({
          amount: 0,
          isFree: true,
          isLegacy: false,
          name: 'Partner Free',
          slug: 'partner-free',
        }),
        plan({
          amount: 0,
          isFree: true,
          isLegacy: true,
          name: 'Legacy Free',
          slug: 'legacy-free',
        }),
        plan({
          amount: 19_900,
          isFree: false,
          isLegacy: false,
          name: 'Pro Plus',
          slug: 'pro-plus',
        }),
      ]),
    ).toEqual({ hobby: null, pro: null });
  });

  it('reports multiple current Pro plans and selects the most expensive', () => {
    const plans = [
      plan({
        amount: 7_500,
        isFree: false,
        isLegacy: false,
        name: 'Pro',
        slug: 'pro-old',
      }),
      plan({
        amount: 9_900,
        isFree: false,
        isLegacy: false,
        name: 'Pro',
        slug: 'pro-new',
      }),
    ];

    expect(pickSelfServePlans(plans).pro).toBe(plans[1]);
    expect(Sentry.captureMessage).toHaveBeenCalledWith(
      'Multiple active, visible Pro billing plans returned',
      {
        level: 'error',
        extra: {
          plans: [
            { amount: 7_500, slug: 'pro-old' },
            { amount: 9_900, slug: 'pro-new' },
          ],
        },
      },
    );
  });

  it('reports missing plans after the backend response loads', () => {
    const plans = [
      plan({
        amount: 0,
        isFree: true,
        isLegacy: false,
        name: 'Hobby',
        slug: 'hobby-from-api',
      }),
    ];

    expect(pickSelfServePlans(plans)).toEqual({
      hobby: plans[0],
      pro: null,
    });
    expect(Sentry.captureMessage).toHaveBeenCalledWith(
      'Missing active, visible Pro billing plan',
      { level: 'error' },
    );
  });
});
