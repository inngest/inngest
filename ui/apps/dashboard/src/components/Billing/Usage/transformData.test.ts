import { describe, expect, it } from 'vitest';

import { createChartOptions } from './transformData';

type Options = ReturnType<typeof createChartOptions>;

function getAxisMax(options: Options) {
  const axis = options.yAxis;
  if (Array.isArray(axis) || typeof axis?.max !== 'function') {
    throw new Error('Expected an allowance-aware axis maximum');
  }
  return axis.max;
}

function getAxisLabelFormatter(options: Options) {
  const axis = options.yAxis;
  const axisLabel = Array.isArray(axis) ? undefined : axis?.axisLabel;
  if (
    !axisLabel ||
    !('formatter' in axisLabel) ||
    typeof axisLabel.formatter !== 'function'
  ) {
    throw new Error('Expected an axis label formatter');
  }
  return axisLabel.formatter;
}

function getMarkAreaBounds(options: Options) {
  const series = options.series;
  if (!Array.isArray(series) || series[0]?.type !== 'bar') {
    throw new Error('Expected a daily bar series');
  }
  const data = series[0].markArea?.data?.[0];
  if (!Array.isArray(data)) {
    throw new Error('Expected a mark area band');
  }
  return data as [MarkAreaBound, MarkAreaBound];
}

type MarkAreaBound = { xAxis: string | number };

const days = (values: number[]) =>
  values.map((value, i) => ({
    time: `2026-09-0${i + 1}T00:00:00Z`,
    value,
  }));

describe('createChartOptions', () => {
  it.each([
    [100, 97, 200],
    [442, 340, 500],
    [50_000, 21_000, 60_000],
    [1_000_000, 0, 2_000_000],
  ])(
    'rounds the axis above limit %i to a whole tick when usage is %i',
    (limit, usage, expected) => {
      const options = createChartOptions(days([usage]), limit, 'execution');

      expect(getAxisMax(options)({ min: 0, max: usage })).toBe(expected);
    },
  );

  it.each([
    [100, 100],
    [100, 445],
  ])(
    'lets the axis auto-scale when usage %i reaches limit %i',
    (limit, usage) => {
      const options = createChartOptions(days([usage]), limit, 'execution');

      expect(getAxisMax(options)({ min: 0, max: usage })).toBeUndefined();
    },
  );

  it('shades from the crossover day to the grid edge', () => {
    const options = createChartOptions(days([60, 60, 60]), 100, 'run');
    const [start, end] = getMarkAreaBounds(options);

    expect(start.xAxis).toBe('2026-09-02T00:00:00.000Z');
    expect(end.xAxis).toBe(Infinity);
  });

  it('shades the last day when the limit is crossed on it', () => {
    const options = createChartOptions(days([40, 40, 40]), 100, 'run');
    const [start, end] = getMarkAreaBounds(options);

    expect(start.xAxis).toBe('2026-09-03T00:00:00.000Z');
    expect(end.xAxis).toBe(Infinity);
  });

  it.each([
    [0, '0'],
    [500, '500'],
    [1000, '1k'],
    [50_000, '50k'],
    [1_000_000, '1m'],
    [3_858_489, '3.86m'],
  ])('formats axis label %i as %s', (value, expected) => {
    const options = createChartOptions(days([value]), Infinity, 'execution');

    expect(getAxisLabelFormatter(options)(value, 0)).toBe(expected);
  });

  it('preserves automatic scaling without limit markers for unlimited plans', () => {
    const options = createChartOptions(days([97]), Infinity, 'execution');

    expect(options.yAxis).toMatchObject({ max: undefined });
    expect(options.series).toMatchObject([
      { markArea: undefined },
      { markLine: undefined },
    ]);
  });
});
