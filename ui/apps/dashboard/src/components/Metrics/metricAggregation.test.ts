import { describe, expect, it } from 'vitest';

import type { ScopedMetric } from '@/gql/graphql';
import {
  latestMetricDataValue,
  sumScopedMetricData,
  ZERO_ID,
} from './metricAggregation';

describe('metric aggregation', () => {
  it('sums scoped metric values by bucket and skips placeholder series', () => {
    const metrics = [
      {
        id: 'fn-1',
        data: [
          { bucket: '2026-06-18T17:00:00.000Z', value: 3 },
          { bucket: '2026-06-18T17:01:00.000Z', value: 4 },
        ],
      },
      {
        id: 'fn-2',
        data: [
          { bucket: '2026-06-18T17:00:00.000Z', value: 7 },
          { bucket: '2026-06-18T17:02:00.000Z', value: 9 },
        ],
      },
      {
        id: ZERO_ID,
        data: [{ bucket: '2026-06-18T17:00:00.000Z', value: 100 }],
      },
    ] satisfies Array<Pick<ScopedMetric, 'data' | 'id'>>;

    expect(sumScopedMetricData(metrics)).toEqual([
      { bucket: '2026-06-18T17:00:00.000Z', value: 10 },
      { bucket: '2026-06-18T17:01:00.000Z', value: 4 },
      { bucket: '2026-06-18T17:02:00.000Z', value: 9 },
    ]);
  });

  it('returns the latest summed metric data value', () => {
    expect(
      latestMetricDataValue([
        { bucket: '2026-06-18T17:02:00.000Z', value: 9 },
        { bucket: '2026-06-18T17:00:00.000Z', value: 10 },
        { bucket: '2026-06-18T17:01:00.000Z', value: 4 },
      ]),
    ).toBe(9);
    expect(latestMetricDataValue()).toBe(0);
  });
});
