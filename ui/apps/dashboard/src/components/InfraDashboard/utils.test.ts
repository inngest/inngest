import { describe, expect, it } from 'vitest';

import { isMenuItemActive } from '@inngest/components/Menu/isMenuItemActive';

import {
  billingCycleDaysRemaining,
  buildTopFunctionRows,
  calculateUsageShare,
  daysUntil,
  formatBytes,
  formatCompactNumber,
  getUtcMonthToDateRange,
  latestBucketMetricTotal,
  latestMetricTotal,
  sumMetricValues,
  sumTimeSeriesValues,
} from './utils';

describe('infra dashboard formatters', () => {
  it('formats compact numbers and bytes', () => {
    expect(formatCompactNumber(42_800_000)).toBe('42.8M');
    expect(formatCompactNumber(312)).toBe('312');
    expect(formatBytes(412 * 1024 ** 3)).toBe('412GB');
    expect(formatBytes(1.4 * 1024 ** 4)).toBe('1.4TB');
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
