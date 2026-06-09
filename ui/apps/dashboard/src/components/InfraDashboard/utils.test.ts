import { describe, expect, it } from 'vitest';

import { isMenuItemActive } from '@inngest/components/Menu/isMenuItemActive';

import {
  billingCycleDaysRemaining,
  buildTopFunctionRows,
  calculateUsageShare,
  daysUntil,
  formatBytes,
  formatCentsMonthly,
  formatCompactNumber,
  getCurrentInfraTierId,
  getInfraPlanBillingAction,
  getUtcMonthToDateRange,
  inferInfraPlanSku,
  latestBucketMetricTotal,
  latestMetricTotal,
  mergeBillingPlanIntoInfraPlans,
  pickInfraConcurrencyAddon,
  sumMetricValues,
  sumTimeSeriesValues,
} from './utils';
import { INFRA_DASHBOARD_PLACEHOLDERS } from './placeholderData';

describe('infra dashboard formatters', () => {
  it('formats compact numbers and bytes', () => {
    expect(formatCompactNumber(42_800_000)).toBe('42.8M');
    expect(formatCompactNumber(312)).toBe('312');
    expect(formatBytes(412 * 1024 ** 3)).toBe('412GB');
    expect(formatBytes(1.4 * 1024 ** 4)).toBe('1.4TB');
    expect(formatCentsMonthly(7_500)).toBe('$75');
  });

  it('calculates billing days remaining', () => {
    expect(
      daysUntil(
        '2026-06-15T00:00:00.000Z',
        new Date('2026-06-08T00:00:00.000Z'),
      ),
    ).toBe(7);
    expect(
      daysUntil('invalid', new Date('2026-06-08T00:00:00.000Z')),
    ).toBeNull();
  });

  it('falls back to UTC end of month for billing cycle days', () => {
    const now = new Date('2026-06-08T00:00:00.000Z');

    expect(billingCycleDaysRemaining('2026-06-15T00:00:00.000Z', now)).toBe(7);
    expect(billingCycleDaysRemaining(null, now)).toBe(23);
    expect(billingCycleDaysRemaining('invalid', now)).toBe(23);
  });

  it('builds UTC month-to-date ranges for billing-backed totals', () => {
    const range = getUtcMonthToDateRange(new Date('2026-06-01T00:30:00.000Z'));

    expect(range.from.toISOString()).toBe('2026-06-01T00:00:00.000Z');
    expect(range.until.toISOString()).toBe('2026-06-01T00:30:00.000Z');
    expect(range.month).toBe(6);
    expect(range.year).toBe(2026);
  });
});

describe('infra dashboard billing plan merge', () => {
  it('infers current infra SKU from plan metadata and concurrency', () => {
    const plans = INFRA_DASHBOARD_PLACEHOLDERS.infraPlans;

    expect(
      inferInfraPlanSku({
        concurrencyLimit: 250,
        defaultSku: 'IN-S',
        plan: null,
        plans,
      }),
    ).toBe('IN-M');
    expect(
      inferInfraPlanSku({
        concurrencyLimit: 300,
        defaultSku: 'IN-S',
        plan: null,
        plans,
      }),
    ).toBe('IN-M');
    expect(
      inferInfraPlanSku({
        concurrencyLimit: 56,
        defaultSku: 'IN-S',
        plan: { isFree: false, name: 'Pro', slug: 'pro-2025-08-08' },
        plans,
      }),
    ).toBe('IN-XS');
  });

  it('merges live concurrency while keeping soft event and queue limits static', () => {
    const result = mergeBillingPlanIntoInfraPlans({
      accountEntitlements: {
        concurrency: { limit: 256 },
        events: { limit: null },
        functionBacklogSize: { limit: 2_500_000 },
      },
      defaultSku: 'IN-S',
      plan: {
        amount: 7_500,
        entitlements: {
          concurrency: { limit: 100 },
          events: { limit: 1_000_000 },
          functionBacklogSize: { limit: 1_000_000 },
        },
        isFree: false,
        name: 'Pro',
        slug: 'pro-2025-08-08',
      },
      plans: INFRA_DASHBOARD_PLACEHOLDERS.infraPlans,
    });

    expect(result.currentPlanSku).toBe('IN-M');
    expect(result.currentPlan).toMatchObject({
      eventStream: '15M events/mo',
      eventStreamLimit: 15_000_000,
      eventStreamUnit: 'events',
      execConcurrency: '256',
      execConcurrencyLimit: 256,
      isCurrent: true,
      priceMonthly: '$75',
      queueDepth: '5M',
      queueDepthLimit: 5_000_000,
      sku: 'IN-M',
    });
    expect(result.plans.find((plan) => plan.sku === 'IN-M')).toEqual(
      result.currentPlan,
    );
    expect(result.plans.find((plan) => plan.sku === 'IN-XS')?.isCurrent).toBe(
      false,
    );
  });

  it('maps current infra SKU to the included infrastructure tier', () => {
    expect(getCurrentInfraTierId('IN-XS')).toBe('free');
    expect(getCurrentInfraTierId('IN-S')).toBe('shared');
    expect(getCurrentInfraTierId('IN-XL')).toBe('shared');
  });
});

describe('infra dashboard billing actions', () => {
  const concurrencyAddon = {
    available: true,
    baseValue: 100,
    maxValue: 1_000,
    name: 'concurrency',
    price: 9_900,
    purchaseCount: 0,
    quantityPer: 100,
  };
  const freePlan = {
    amount: 0,
    isFree: true,
    name: 'Hobby',
    slug: 'hobby-free-2025-08-08',
  };
  const proPlan = {
    amount: 7_500,
    isFree: false,
    name: 'Pro',
    slug: 'pro-2025-08-08',
  };

  it('opens Pro checkout when a free account selects IN-S', () => {
    expect(
      getInfraPlanBillingAction({
        concurrencyAddon,
        currentConcurrencyLimit: 5,
        currentPlan: freePlan,
        currentPlanSku: 'IN-XS',
        targetSku: 'IN-S',
      }),
    ).toEqual({
      addonUpdate: null,
      item: {
        amount: 7_500,
        name: 'Pro',
        planSlug: 'pro-2025-08-08',
        quantity: 1,
      },
      type: 'upgrade-base-plan',
    });
  });

  it('does not require addon metadata when selecting base Pro', () => {
    expect(
      getInfraPlanBillingAction({
        concurrencyAddon: null,
        currentConcurrencyLimit: 5,
        currentPlan: freePlan,
        currentPlanSku: 'IN-XS',
        targetSku: 'IN-S',
      }),
    ).toMatchObject({
      addonUpdate: null,
      type: 'upgrade-base-plan',
    });
  });

  it('opens Pro checkout with a follow-up addon quantity for IN-L', () => {
    expect(
      getInfraPlanBillingAction({
        concurrencyAddon,
        currentConcurrencyLimit: 5,
        currentPlan: freePlan,
        currentPlanSku: 'IN-XS',
        targetSku: 'IN-L',
      }),
    ).toMatchObject({
      addonUpdate: {
        addonName: 'concurrency',
        addonQuantity: 4,
        targetMonthlyAmountCents: 59_900,
        targetConcurrency: 500,
        targetSku: 'IN-L',
      },
      type: 'upgrade-base-plan',
    });
  });

  it('updates concurrency add-on quantity when Pro selects a larger SKU', () => {
    expect(
      getInfraPlanBillingAction({
        concurrencyAddon,
        currentConcurrencyLimit: 100,
        currentPlan: proPlan,
        currentPlanSku: 'IN-S',
        targetSku: 'IN-M',
      }),
    ).toEqual({
      addonName: 'concurrency',
      addonQuantity: 2,
      estimatedMonthlyAddonCost: 19_800,
      isIncrease: true,
      targetConcurrency: 250,
      targetMonthlyAmountCents: 24_900,
      targetSku: 'IN-M',
      type: 'update-concurrency-addon',
    });
  });

  it('calculates addon quantity from Pro base concurrency instead of addon base value', () => {
    expect(
      getInfraPlanBillingAction({
        concurrencyAddon: { ...concurrencyAddon, baseValue: 0 },
        currentConcurrencyLimit: 100,
        currentPlan: proPlan,
        currentPlanSku: 'IN-S',
        targetSku: 'IN-M',
      }),
    ).toMatchObject({
      addonQuantity: 2,
      targetConcurrency: 250,
      targetMonthlyAmountCents: 24_900,
      type: 'update-concurrency-addon',
    });
  });

  it('decreases concurrency add-on quantity back to base Pro when selecting IN-S', () => {
    expect(
      getInfraPlanBillingAction({
        concurrencyAddon,
        currentConcurrencyLimit: 500,
        currentPlan: proPlan,
        currentPlanSku: 'IN-L',
        targetSku: 'IN-S',
      }),
    ).toEqual({
      addonName: 'concurrency',
      addonQuantity: 0,
      estimatedMonthlyAddonCost: 0,
      isIncrease: false,
      targetConcurrency: 100,
      targetMonthlyAmountCents: 9_900,
      targetSku: 'IN-S',
      type: 'update-concurrency-addon',
    });
  });

  it('can remove concurrency addons without loaded addon pricing metadata', () => {
    expect(
      getInfraPlanBillingAction({
        concurrencyAddon: null,
        currentConcurrencyLimit: 500,
        currentPlan: proPlan,
        currentPlanSku: 'IN-L',
        targetSku: 'IN-S',
      }),
    ).toEqual({
      addonName: 'concurrency',
      addonQuantity: 0,
      estimatedMonthlyAddonCost: 0,
      isIncrease: false,
      targetConcurrency: 100,
      targetMonthlyAmountCents: 9_900,
      targetSku: 'IN-S',
      type: 'update-concurrency-addon',
    });
  });

  it('removes concurrency addons before canceling to free', () => {
    expect(
      getInfraPlanBillingAction({
        concurrencyAddon,
        currentConcurrencyLimit: 500,
        currentPlan: proPlan,
        currentPlanSku: 'IN-L',
        targetSku: 'IN-XS',
      }),
    ).toEqual({
      addonUpdate: {
        addonName: 'concurrency',
        addonQuantity: 0,
        estimatedMonthlyAddonCost: 0,
        isIncrease: false,
        targetConcurrency: 5,
        targetMonthlyAmountCents: 0,
        targetSku: 'IN-XS',
      },
      item: {
        amount: 0,
        name: 'Hobby',
        planSlug: 'hobby-free-2025-08-08',
        quantity: 1,
      },
      type: 'cancel-to-free',
    });
  });

  it('uses add-on metadata even when the account availability flag is false', () => {
    expect(
      pickInfraConcurrencyAddon({
        accountAddon: { ...concurrencyAddon, available: false },
        planAddon: concurrencyAddon,
      }),
    ).toEqual({ ...concurrencyAddon, available: false });
  });

  it('falls back to plan addon metadata when account addon sizing is incomplete', () => {
    expect(
      pickInfraConcurrencyAddon({
        accountAddon: { ...concurrencyAddon, price: null },
        planAddon: concurrencyAddon,
      }),
    ).toBe(concurrencyAddon);
  });

  it('treats the floored SKU as current', () => {
    expect(
      getInfraPlanBillingAction({
        concurrencyAddon,
        currentConcurrencyLimit: 300,
        currentPlan: proPlan,
        currentPlanSku: 'IN-M',
        targetSku: 'IN-M',
      }),
    ).toEqual({ type: 'current' });
  });

  it('returns unavailable for missing addon data or targets above addon max', () => {
    expect(
      getInfraPlanBillingAction({
        concurrencyAddon: { ...concurrencyAddon, price: null },
        currentConcurrencyLimit: 100,
        currentPlan: proPlan,
        currentPlanSku: 'IN-S',
        targetSku: 'IN-M',
      }),
    ).toMatchObject({ type: 'unavailable' });
    expect(
      getInfraPlanBillingAction({
        concurrencyAddon: { ...concurrencyAddon, maxValue: 500 },
        currentConcurrencyLimit: 500,
        currentPlan: proPlan,
        currentPlanSku: 'IN-L',
        targetSku: 'IN-XL',
      }),
    ).toMatchObject({ type: 'unavailable' });
  });
});

describe('infra dashboard metric aggregation', () => {
  it('sums all metric points and latest points', () => {
    const metrics = [
      { data: [{ value: 1 }, { value: 2 }] },
      { data: [{ value: 3 }, { value: 4 }] },
    ];

    expect(sumMetricValues(metrics)).toBe(10);
    expect(latestMetricTotal(metrics)).toBe(6);
  });

  it('sums current backlog from the newest metric bucket', () => {
    const metrics = [
      {
        data: [
          { bucket: '2026-06-09T10:00:00.000Z', value: 100 },
          { bucket: '2026-06-09T10:01:00.000Z', value: 75 },
        ],
      },
      {
        data: [
          { bucket: '2026-06-09T10:00:00.000Z', value: 25 },
          { bucket: '2026-06-09T10:01:00.000Z', value: 10 },
        ],
      },
      {
        data: [{ bucket: '2026-06-09T09:59:00.000Z', value: 500 }],
      },
    ];

    expect(latestBucketMetricTotal(metrics)).toBe(85);
  });

  it('calculates usage share safely', () => {
    expect(calculateUsageShare(30, 100)).toBe(30);
    expect(calculateUsageShare(1, 3)).toBe(33.3);
    expect(calculateUsageShare(1, 0)).toBe(0);
  });

  it('sums nullable billing time series points', () => {
    const series = [
      { data: [{ value: 12 }, { value: null }] },
      { data: [{ value: 8 }, { value: 5 }] },
    ];

    expect(sumTimeSeriesValues(series)).toBe(25);
    expect(sumTimeSeriesValues(undefined)).toBe(0);
  });
});

describe('infra dashboard top functions', () => {
  it('joins function usage to app/function names and sorts by runs', () => {
    const functions = [
      {
        app: { externalID: 'app-1', name: 'App 1' },
        id: 'fn-1',
        isArchived: false,
        isPaused: false,
        name: 'Function 1',
        slug: 'function-1',
        triggers: [],
      },
      {
        app: { externalID: 'app-2', name: 'App 2' },
        id: 'fn-2',
        isArchived: false,
        isPaused: false,
        name: 'Function 2',
        slug: 'function-2',
        triggers: [],
      },
    ] as unknown as NonNullable<
      Parameters<typeof buildTopFunctionRows>[0]['functions']
    >;
    const usage = [
      {
        id: 'fn-1',
        slug: 'function-1',
        dailyStarts: { total: 10, data: [] },
        dailyCompleted: { total: 8, data: [] },
        dailyCancelled: { total: 1, data: [] },
        dailyFailures: { total: 1, data: [] },
      },
      {
        id: 'fn-2',
        slug: 'function-2',
        dailyStarts: { total: 30, data: [] },
        dailyCompleted: { total: 25, data: [] },
        dailyCancelled: { total: 2, data: [] },
        dailyFailures: { total: 3, data: [] },
      },
    ] as unknown as NonNullable<
      Parameters<typeof buildTopFunctionRows>[0]['usage']
    >;

    expect(buildTopFunctionRows({ functions, usage })).toEqual([
      {
        app: {
          externalID: 'app-2',
          name: 'App 2',
        },
        failureRate: 10,
        id: 'fn-2',
        isArchived: false,
        isPaused: false,
        name: 'Function 2',
        slug: 'function-2',
        triggers: [],
        usage: {
          dailyVolumeSlots: [],
          totalVolume: 30,
        },
      },
      {
        app: {
          externalID: 'app-1',
          name: 'App 1',
        },
        failureRate: 10,
        id: 'fn-1',
        isArchived: false,
        isPaused: false,
        name: 'Function 1',
        slug: 'function-1',
        triggers: [],
        usage: {
          dailyVolumeSlots: [],
          totalVolume: 10,
        },
      },
    ]);
  });

  it('builds sorted top rows from enriched usage and does not hard-cap at five', () => {
    const usage = [10, 60, 20, 50, 30, 40].map((total, index) => ({
      app: { externalID: `app-${index + 1}`, name: `App ${index + 1}` },
      id: `fn-${index + 1}`,
      isArchived: false,
      isPaused: false,
      name: `Function ${index + 1}`,
      slug: `function-${index + 1}`,
      triggers: [],
      dailyStarts: { total, data: [] },
      dailyCompleted: { total, data: [] },
      dailyCancelled: { total: 0, data: [] },
      dailyFailures: { total: 0, data: [] },
    })) as unknown as NonNullable<
      Parameters<typeof buildTopFunctionRows>[0]['usage']
    >;

    const rows = buildTopFunctionRows({ functions: undefined, usage });

    expect(rows).toHaveLength(6);
    expect(rows.map((row) => row.name)).toEqual([
      'Function 2',
      'Function 4',
      'Function 6',
      'Function 5',
      'Function 3',
      'Function 1',
    ]);
  });

  it('applies the display limit after sorting by run volume', () => {
    const usage = [10, 60, 20, 50, 30, 40].map((total, index) => ({
      app: { externalID: `app-${index + 1}`, name: `App ${index + 1}` },
      id: `fn-${index + 1}`,
      isArchived: false,
      isPaused: false,
      name: `Function ${index + 1}`,
      slug: `function-${index + 1}`,
      triggers: [],
      dailyStarts: { total, data: [] },
      dailyCompleted: { total, data: [] },
      dailyCancelled: { total: 0, data: [] },
      dailyFailures: { total: 0, data: [] },
    })) as unknown as NonNullable<
      Parameters<typeof buildTopFunctionRows>[0]['usage']
    >;

    const rows = buildTopFunctionRows({
      functions: undefined,
      limit: 3,
      usage,
    });

    expect(rows.map((row) => row.name)).toEqual([
      'Function 2',
      'Function 4',
      'Function 6',
    ]);
  });

  it('filters functions with no runs from top rows', () => {
    const usage = [0, 12, 0, 4].map((total, index) => ({
      app: { externalID: `app-${index + 1}`, name: `App ${index + 1}` },
      id: `fn-${index + 1}`,
      isArchived: false,
      isPaused: false,
      name: `Function ${index + 1}`,
      slug: `function-${index + 1}`,
      triggers: [],
      dailyStarts: { total, data: [] },
      dailyCompleted: { total, data: [] },
      dailyCancelled: { total: 0, data: [] },
      dailyFailures: { total: 0, data: [] },
    })) as unknown as NonNullable<
      Parameters<typeof buildTopFunctionRows>[0]['usage']
    >;

    const rows = buildTopFunctionRows({ functions: undefined, usage });

    expect(rows.map((row) => row.name)).toEqual(['Function 2', 'Function 4']);
  });
});

describe('MenuItem exact matching', () => {
  it('matches dashboard routes exactly while ignoring query and trailing slash', () => {
    expect(
      isMenuItemActive('/env/production/?nav=v2', '/env/production', true),
    ).toBe(true);
    expect(
      isMenuItemActive('/env/production/apps', '/env/production', true),
    ).toBe(false);
    expect(
      isMenuItemActive('/env/production/apps', '/env/production', false),
    ).toBe(true);
  });
});
