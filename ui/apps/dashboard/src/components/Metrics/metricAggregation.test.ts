import { describe, expect, it } from 'vitest';

import type { ScopedMetric } from '@/gql/graphql';
import {
  latestMetricDataValue,
  sumScopedMetricData,
  sumScopedMetricsByGroup,
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

  it('sums scoped metric values by group and fills missing buckets with zero', () => {
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
        id: 'fn-3',
        data: [{ bucket: '2026-06-18T17:01:00.000Z', value: 5 }],
      },
      {
        id: 'fn-without-app',
        data: [{ bucket: '2026-06-18T17:00:00.000Z', value: 100 }],
      },
      {
        id: ZERO_ID,
        data: [{ bucket: '2026-06-18T17:00:00.000Z', value: 100 }],
      },
    ] satisfies Array<Pick<ScopedMetric, 'data' | 'id'>>;

    const appIDsByFunctionID: Record<string, string | undefined> = {
      'fn-1': 'app-a',
      'fn-2': 'app-a',
      'fn-3': 'app-b',
    };

    expect(
      sumScopedMetricsByGroup(metrics, ({ id }) => appIDsByFunctionID[id]),
    ).toEqual([
      {
        id: 'app-a',
        tagName: null,
        tagValue: null,
        data: [
          { bucket: '2026-06-18T17:00:00.000Z', value: 10 },
          { bucket: '2026-06-18T17:01:00.000Z', value: 4 },
          { bucket: '2026-06-18T17:02:00.000Z', value: 9 },
        ],
      },
      {
        id: 'app-b',
        tagName: null,
        tagValue: null,
        data: [
          { bucket: '2026-06-18T17:00:00.000Z', value: 0 },
          { bucket: '2026-06-18T17:01:00.000Z', value: 5 },
          { bucket: '2026-06-18T17:02:00.000Z', value: 0 },
        ],
      },
    ]);
  });
});
