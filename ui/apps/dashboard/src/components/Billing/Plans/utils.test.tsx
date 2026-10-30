import * as Sentry from '@sentry/tanstackstart-react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  displayPlanName,
  executionUsageDescription,
  isActive,
  pickSelfServePlans,
  processPlan,
  type SelfServePlan,
} from './utils';

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
    addons: {
      concurrency: {
        available: false,
        baseValue: null,
        maxValue: 100_000,
        name: 'concurrency',
        price: null,
        purchaseCount: 0,
        quantityPer: 25,
      },
    },
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
        amount: 19_900,
        isFree: false,
        isLegacy: false,
        name: 'Business',
        slug: 'business-from-api',
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
      business: plans[2],
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
        plan({
          amount: 0,
          isFree: true,
          isLegacy: false,
          name: 'Growth',
          slug: 'free-growth',
        }),
        plan({
          amount: 19_900,
          isFree: false,
          isLegacy: true,
          name: 'Growth',
          slug: 'retired-growth',
        }),
      ]),
    ).toEqual({ hobby: null, pro: null, business: null });
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

  it('reports multiple current Growth plans and selects the most expensive', () => {
    const plans = [
      plan({
        amount: 15_000,
        isFree: false,
        isLegacy: false,
        name: 'Growth',
        slug: 'growth-old',
      }),
      plan({
        amount: 19_900,
        isFree: false,
        isLegacy: false,
        name: 'Growth',
        slug: 'growth-new',
      }),
    ];

    expect(pickSelfServePlans(plans).business).toBe(plans[1]);
    expect(Sentry.captureMessage).toHaveBeenCalledWith(
      'Multiple active, visible Business billing plans returned',
      {
        level: 'error',
        extra: {
          plans: [
            { amount: 15_000, slug: 'growth-old' },
            { amount: 19_900, slug: 'growth-new' },
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
      business: null,
    });
    expect(Sentry.captureMessage).toHaveBeenCalledWith(
      'Missing active, visible Pro billing plan',
      { level: 'error' },
    );
    expect(Sentry.captureMessage).toHaveBeenCalledWith(
      'Missing active, visible Business billing plan',
      { level: 'error' },
    );
  });

  it('uses Growth only when Business is unavailable', () => {
    const growth = plan({
      amount: 19_900,
      isFree: false,
      isLegacy: false,
      name: 'Growth',
      slug: 'growth-from-api',
    });
    const business = plan({
      amount: 19_900,
      isFree: false,
      isLegacy: false,
      name: 'Business',
      slug: 'business-from-api',
    });

    expect(pickSelfServePlans([growth]).business).toBe(growth);
    expect(Sentry.captureMessage).not.toHaveBeenCalledWith(
      'Missing active, visible Business billing plan',
      { level: 'error' },
    );
    expect(pickSelfServePlans([growth, business]).business).toBe(business);
  });

  it('reports an active Growth plan next to an active Business plan', () => {
    const growth = plan({
      amount: 19_900,
      isFree: false,
      isLegacy: false,
      name: 'Growth',
      slug: 'growth-from-api',
    });
    const business = plan({
      amount: 15_000,
      isFree: false,
      isLegacy: false,
      name: 'Business',
      slug: 'business-from-api',
    });

    expect(pickSelfServePlans([growth, business]).business).toBe(business);
    expect(Sentry.captureMessage).toHaveBeenCalledWith(
      'Multiple active, visible Business billing plans returned',
      {
        level: 'error',
        extra: {
          plans: [
            { amount: 19_900, slug: 'growth-from-api' },
            { amount: 15_000, slug: 'business-from-api' },
          ],
        },
      },
    );
  });
});

describe('Business plan display', () => {
  it('shows the new name for a Growth plan without changing its slug', () => {
    const growth = plan({
      amount: 19_900,
      isFree: false,
      isLegacy: false,
      name: 'Growth',
      slug: 'growth-from-api',
    });

    expect(displayPlanName(growth.name)).toBe('Business');
    expect(processPlan(growth).name).toBe('Business');
    expect(processPlan(growth).slug).toBe('growth-from-api');
    expect(displayPlanName('Business')).toBe('Business');
  });

  it('keeps the suffix of a Growth plan name', () => {
    expect(displayPlanName('Growth Trial')).toBe('Business Trial');
    expect(displayPlanName('Growth - Annual')).toBe('Business - Annual');
  });

  it('matches a Growth current plan to the Business card', () => {
    expect(isActive({ name: 'Growth' }, { name: 'Business' })).toBe(true);
    expect(isActive({ name: 'Pro' }, { name: 'Business' })).toBe(false);
  });
});

describe('execution usage description', () => {
  it.each([
    ['Pro', 1_000_000, 50, '1,000,000'],
    ['Business', 10_000_000, 25, '10,000,000'],
    ['Growth', 1_000_000, 50, '1,000,000'],
    ['Pro', 4_999_999, 50, '4,999,999'],
    ['Pro', 5_000_000, 25, '5,000,000'],
    ['Pro', 14_999_999, 25, '14,999,999'],
    ['Pro', 15_000_000, 20, '15,000,000'],
    ['Pro', 49_999_999, 20, '49,999,999'],
    ['Pro', 50_000_000, 15, '50,000,000'],
  ])(
    'uses the allowance for %s with %i included',
    (name, limit, price, formattedLimit) => {
      expect(
        executionUsageDescription({ name, slug: 'paid-plan' }, limit),
      ).toBe(
        `Usage beyond ${formattedLimit} is billed at $${price} per 1,000,000 at the start of the next billing cycle.`,
      );
    },
  );

  it.each([
    ['Enterprise', 1_000_000],
    ['Enterprise - Annual', 10_000_000],
    ['enterprise custom', 50_000_000],
    ['Enterprise', null],
    ['Pro', null],
  ])('keeps the existing copy for %s with limit %s', (name, limit) => {
    expect(executionUsageDescription({ name, slug: 'plan' }, limit)).toBe(
      'Additional usage billed at the start of the next billing cycle.',
    );
  });

  it('keeps the free Hobby execution limit copy', () => {
    expect(
      executionUsageDescription(
        { name: 'Hobby', slug: 'hobby-free-2025-08-08' },
        50_000,
      ),
    ).toBe('The maximum number of executions per month');
  });
});
