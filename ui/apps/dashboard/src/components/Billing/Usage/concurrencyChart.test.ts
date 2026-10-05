import { describe, expect, it } from 'vitest';

import { mergeLimitHitRanges } from './concurrencyChart';

const hour = (n: number) => `2026-09-01T${String(n).padStart(2, '0')}:00:00Z`;
const ms = (iso: string) => new Date(iso).getTime();

describe('mergeLimitHitRanges', () => {
  it('merges adjacent hours into one range ending an hour after the last bucket', () => {
    const ranges = mergeLimitHitRanges([
      { bucket: hour(0), value: 0 },
      { bucket: hour(1), value: 3 },
      { bucket: hour(2), value: 1 },
      { bucket: hour(3), value: 0 },
      { bucket: hour(5), value: 2 },
      { bucket: hour(6), value: null },
    ]);

    expect(ranges).toEqual([
      [{ xAxis: ms(hour(1)) }, { xAxis: ms(hour(3)) }],
      [{ xAxis: ms(hour(5)) }, { xAxis: ms(hour(6)) }],
    ]);
  });

  it('does not merge hits whose buckets are not consecutive', () => {
    const ranges = mergeLimitHitRanges([
      { bucket: hour(4), value: 1 },
      { bucket: hour(1), value: 1 },
    ]);

    expect(ranges).toHaveLength(2);
  });
});
