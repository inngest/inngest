import { describe, expect, it } from 'vitest';

import {
  createBacklogChartOptions,
  createConcurrencyChartOptions,
  HOUR_MS,
  mergeHourRanges,
  niceCeiling,
  type ConcurrencyTimeSeries,
} from './concurrencyChart';

const hour = (n: number) => `2026-09-01T${String(n).padStart(2, '0')}:00:00Z`;
const ms = (iso: string) => new Date(iso).getTime();

describe('mergeHourRanges', () => {
  it('merges consecutive matching hours into ranges ending an hour after the last bucket', () => {
    const ranges = mergeHourRanges(
      [
        { bucket: hour(0), value: 5 },
        { bucket: hour(1), value: 12 },
        { bucket: hour(2), value: 11 },
        { bucket: hour(3), value: 10 },
        { bucket: hour(5), value: 20 },
        { bucket: hour(6), value: null },
      ],
      (value) => value > 10,
    );

    expect(ranges).toEqual([
      [{ xAxis: ms(hour(1)) }, { xAxis: ms(hour(3)) }],
      [{ xAxis: ms(hour(5)) }, { xAxis: ms(hour(6)) }],
    ]);
  });
});

describe('niceCeiling', () => {
  it.each([
    [29, 40],
    [60, 80],
    [5, 10],
  ])('rounds %d up to the clean tick ceiling %d', (value, expected) => {
    expect(niceCeiling(value)).toBe(expected);
  });
});

describe('createConcurrencyChartOptions', () => {
  const threeHourBandRangeHours = 300 * 3;
  const at = (h: number) => ms(hour(0)) + h * HOUR_MS;
  const burstsAt = (hours: number[]) =>
    hours.map((h) => ({ bucket: new Date(at(h)).toISOString(), value: 8 }));

  const limitSeriesData = (
    buckets: ConcurrencyTimeSeries['buckets'],
    rangeHours: number,
  ) => {
    const series: ConcurrencyTimeSeries = {
      baseLimit: 5,
      burstLimit: 10,
      buckets,
      limitHits: [],
    };
    const range = {
      from: hour(0),
      to: new Date(at(rangeHours)).toISOString(),
    };
    const options = createConcurrencyChartOptions(series, range);
    const seriesList = options.series as { name: string; data: unknown }[];
    return seriesList.find((s) => s.name === 'Concurrency limit')?.data;
  };

  it('raises the limit line to the burst limit during a burst', () => {
    const data = limitSeriesData(
      [
        { bucket: hour(1), value: 3 },
        { bucket: hour(2), value: 8 },
        { bucket: hour(3), value: 4 },
      ],
      24,
    );

    expect(data).toEqual([
      [at(0), 5],
      [at(2), 5],
      [at(2), 10],
      [at(3), 10],
      [at(3), 5],
      [at(24), 5],
    ]);
  });

  it('raises the limit line once across bursts whose widened bands overlap', () => {
    const data = limitSeriesData(burstsAt([5, 7]), threeHourBandRangeHours);

    expect(data).toEqual([
      [at(0), 5],
      [at(4), 5],
      [at(4), 10],
      [at(9), 10],
      [at(9), 5],
      [at(threeHourBandRangeHours), 5],
    ]);
  });

  it('keeps widened bursts at the range edges inside the range', () => {
    const lastHour = threeHourBandRangeHours - 1;
    const data = limitSeriesData(
      burstsAt([0, lastHour]),
      threeHourBandRangeHours,
    );

    expect(data).toEqual([
      [at(0), 5],
      [at(0), 5],
      [at(0), 10],
      [at(3), 10],
      [at(3), 5],
      [at(lastHour - 2), 5],
      [at(lastHour - 2), 10],
      [at(threeHourBandRangeHours), 10],
      [at(threeHourBandRangeHours), 5],
      [at(threeHourBandRangeHours), 5],
    ]);
  });
});

describe('createBacklogChartOptions', () => {
  const at = (h: number) => ms(hour(0)) + h * HOUR_MS;
  const range = { from: hour(0), to: new Date(at(24)).toISOString() };
  const bands = {
    baseLimit: 50,
    buckets: [
      { bucket: hour(1), value: 80 },
      { bucket: hour(2), value: 10 },
    ],
    limitHits: [
      { bucket: hour(1), value: 0 },
      { bucket: hour(2), value: 3 },
    ],
  };
  const options = createBacklogChartOptions(
    { buckets: [{ bucket: hour(1), value: 3 }] },
    bands,
    range,
  );
  const seriesList = options.series as {
    name: string;
    markArea?: { data: unknown };
  }[];

  it('sizes the y axis from the backlog alone, ignoring the limits', () => {
    expect((options.yAxis as { max: number }).max).toBe(10);
  });

  it('draws a plain line with no limit line', () => {
    expect(seriesList.map((s) => s.name)).toEqual([
      'Concurrency burst',
      'Concurrency hit',
      'Backlog',
    ]);
  });

  it('places the bands on the burst and limit-hit hours', () => {
    expect(
      seriesList.find((s) => s.name === 'Concurrency burst')?.markArea?.data,
    ).toEqual([[{ xAxis: at(1) }, { xAxis: at(2) }]]);
    expect(
      seriesList.find((s) => s.name === 'Concurrency hit')?.markArea?.data,
    ).toEqual([[{ xAxis: at(2) }, { xAxis: at(3) }]]);
  });
});
