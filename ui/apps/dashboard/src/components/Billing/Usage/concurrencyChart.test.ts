import { describe, expect, it } from 'vitest';

import { mergeHourRanges, niceCeiling } from './concurrencyChart';

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
