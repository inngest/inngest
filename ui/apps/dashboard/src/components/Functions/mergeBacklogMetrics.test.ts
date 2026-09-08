import { describe, expect, test } from 'vitest';

import { mergeBacklogMetrics } from './mergeBacklogMetrics';

describe('mergeBacklogMetrics', () => {
  test('joins sparse series by timestamp without inventing zeroes', () => {
    expect(
      mergeBacklogMetrics(
        [
          { bucket: '2026-08-23T03:16:00Z', value: 144650 },
          { bucket: '2026-08-23T03:18:00Z', value: 144607 },
          { bucket: '2026-08-23T03:20:00Z', value: 0 },
        ],
        [
          { bucket: '2026-08-23T03:17:00Z', value: 12 },
          { bucket: '2026-08-23T03:18:00Z', value: 10 },
        ],
        '1m',
        '2026-08-23T03:16:00Z',
        '2026-08-23T03:21:00Z',
      ),
    ).toEqual([
      {
        name: '2026-08-23T03:16:00Z',
        values: { scheduled: 144650 },
      },
      {
        name: '2026-08-23T03:17:00Z',
        values: { sleeping: 12 },
      },
      {
        name: '2026-08-23T03:18:00Z',
        values: { scheduled: 144607, sleeping: 10 },
      },
      {
        name: '2026-08-23T03:19:00.000Z',
        values: {},
      },
      {
        name: '2026-08-23T03:20:00Z',
        values: { scheduled: 0 },
      },
    ]);
  });

  test('preserves null values as chart gaps', () => {
    expect(
      mergeBacklogMetrics(
        [
          { bucket: '2026-08-23T03:16:00Z', value: null },
          { bucket: '2026-08-23T03:17:00Z', value: 2 },
        ],
        [],
        '1m',
        '2026-08-23T03:16:00Z',
        '2026-08-23T03:18:00Z',
      ),
    ).toEqual([
      { name: '2026-08-23T03:16:00Z', values: { scheduled: null } },
      { name: '2026-08-23T03:17:00Z', values: { scheduled: 2 } },
    ]);
  });

  test('shows gaps lasting at least three minutes as inferred zeroes', () => {
    expect(
      mergeBacklogMetrics(
        [
          { bucket: '2026-08-23T03:16:00Z', value: 144650 },
          { bucket: '2026-08-23T03:20:00Z', value: 144607 },
        ],
        [],
        '1m',
        '2026-08-23T03:13:00Z',
        '2026-08-23T03:24:30Z',
      ),
    ).toEqual([
      {
        name: '2026-08-23T03:13:00.000Z',
        values: { scheduled: 0 },
        inferred: ['scheduled'],
      },
      {
        name: '2026-08-23T03:14:00.000Z',
        values: { scheduled: 0 },
        inferred: ['scheduled'],
      },
      {
        name: '2026-08-23T03:15:00.000Z',
        values: { scheduled: 0 },
        inferred: ['scheduled'],
      },
      {
        name: '2026-08-23T03:16:00Z',
        values: { scheduled: 144650 },
      },
      {
        name: '2026-08-23T03:17:00.000Z',
        values: { scheduled: 0 },
        inferred: ['scheduled'],
      },
      {
        name: '2026-08-23T03:18:00.000Z',
        values: { scheduled: 0 },
        inferred: ['scheduled'],
      },
      {
        name: '2026-08-23T03:19:00.000Z',
        values: { scheduled: 0 },
        inferred: ['scheduled'],
      },
      {
        name: '2026-08-23T03:20:00Z',
        values: { scheduled: 144607 },
      },
      {
        name: '2026-08-23T03:21:00.000Z',
        values: { scheduled: 0 },
        inferred: ['scheduled'],
      },
      {
        name: '2026-08-23T03:22:00.000Z',
        values: { scheduled: 0 },
        inferred: ['scheduled'],
      },
      {
        name: '2026-08-23T03:23:00.000Z',
        values: { scheduled: 0 },
        inferred: ['scheduled'],
      },
    ]);
  });

  test('leaves gaps shorter than three minutes and empty responses blank', () => {
    expect(
      mergeBacklogMetrics(
        [
          { bucket: '2026-08-23T03:16:00Z', value: 4 },
          { bucket: '2026-08-23T03:18:00Z', value: 3 },
          { bucket: '2026-08-23T03:20:00Z', value: 2 },
        ],
        [],
        '1m',
        '2026-08-23T03:14:00Z',
        '2026-08-23T03:23:30Z',
      ).map(({ name, values }) => ({ name, values })),
    ).toEqual([
      { name: '2026-08-23T03:16:00Z', values: { scheduled: 4 } },
      { name: '2026-08-23T03:17:00.000Z', values: {} },
      { name: '2026-08-23T03:18:00Z', values: { scheduled: 3 } },
      { name: '2026-08-23T03:19:00.000Z', values: {} },
      { name: '2026-08-23T03:20:00Z', values: { scheduled: 2 } },
    ]);
    expect(
      mergeBacklogMetrics(
        [],
        [],
        '1m',
        '2026-08-23T03:00:00Z',
        '2026-08-23T04:00:00Z',
      ),
    ).toEqual([]);
  });

  test('uses elapsed time rather than bucket count for the threshold', () => {
    expect(
      mergeBacklogMetrics(
        [
          { bucket: '2026-08-23T03:10:00Z', value: 4 },
          { bucket: '2026-08-23T03:20:00Z', value: 2 },
        ],
        [],
        '5m',
        '2026-08-23T03:10:00Z',
        '2026-08-23T03:25:00Z',
      ),
    ).toEqual([
      { name: '2026-08-23T03:10:00Z', values: { scheduled: 4 } },
      {
        name: '2026-08-23T03:15:00.000Z',
        values: { scheduled: 0 },
        inferred: ['scheduled'],
      },
      { name: '2026-08-23T03:20:00Z', values: { scheduled: 2 } },
    ]);
  });
});
