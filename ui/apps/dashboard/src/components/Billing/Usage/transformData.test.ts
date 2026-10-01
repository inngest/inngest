import { describe, expect, it } from 'vitest';

import { createChartOptions } from './transformData';

describe('createChartOptions', () => {
  it.each([0, 97, 100, 150])(
    'keeps the allowance and usage visible at %i%% usage',
    (usage) => {
      const options = createChartOptions(
        [{ time: '2026-09-01T00:00:00Z', value: usage }],
        100,
        'execution',
      );
      const axis = options.yAxis;
      if (Array.isArray(axis) || typeof axis?.max !== 'function') {
        throw new Error('Expected an allowance-aware axis maximum');
      }

      const maximum = axis.max({ min: 0, max: usage });
      expect(maximum).toBeGreaterThan(100);
      expect(maximum).toBeGreaterThan(usage);
      expect(options.series).toMatchObject([
        {},
        {
          markLine: { data: [{ yAxis: 100 }] },
          markArea: usage >= 100 ? expect.any(Object) : undefined,
        },
      ]);
    },
  );

  it.each([
    [100, 97, 110],
    [445, 340, 490],
    [100, 445, 490],
  ])(
    'rounds the axis maximum for limit %i and usage %i',
    (limit, usage, expected) => {
      const options = createChartOptions(
        [{ time: '2026-09-01T00:00:00Z', value: usage }],
        limit,
        'execution',
      );
      const axis = options.yAxis;
      if (Array.isArray(axis) || typeof axis?.max !== 'function') {
        throw new Error('Expected an allowance-aware axis maximum');
      }

      expect(axis.max({ min: 0, max: usage })).toBe(expected);
    },
  );

  it('preserves automatic scaling without limit markers for unlimited plans', () => {
    const options = createChartOptions(
      [{ time: '2026-09-01T00:00:00Z', value: 97 }],
      Infinity,
      'execution',
    );

    expect(options.yAxis).toMatchObject({ max: undefined });
    expect(options.series).toMatchObject([
      {},
      { markLine: undefined, markArea: undefined },
    ]);
  });
});
