import { describe, expect, it } from 'vitest';

import type { ScopedMetric } from '@/gql/graphql';
import { mapEntityLines } from './utils';

const bucket = '2026-10-28T16:00:00Z';
const metric = (
  id: string,
  tagValue: string | null,
  value = 1,
): ScopedMetric => ({
  id,
  tagName: 'status',
  tagValue,
  data: [{ bucket, value }],
});

const entities = {
  'fn-a': { id: 'fn-a', name: 'Send email' },
  'fn-b': { id: 'fn-b', name: 'Create user' },
};

describe('mapEntityLines', () => {
  it('labels failed series with function names and keeps their counts attached', () => {
    const result = mapEntityLines(
      [metric('fn-a', 'Failed', 3), metric('fn-b', 'Failed', 7)],
      entities,
    );

    expect(result.legendData).toEqual(['Create user', 'Send email']);
    expect(result.series.map(({ name, data }) => ({ name, data }))).toEqual([
      { name: 'Create user', data: [7] },
      { name: 'Send email', data: [3] },
    ]);
  });

  it.each(['Failed', null])(
    'uses the entity ID when its name is missing (%s)',
    (tagValue) => {
      expect(
        mapEntityLines([metric('fn-missing', tagValue)], {}).series[0]?.name,
      ).toBe('fn-missing');
    },
  );

  it('preserves worker labels when explicitly naming series by tag', () => {
    const result = mapEntityLines(
      [metric('fn-a', 'worker-2'), metric('fn-a', 'worker-1')],
      entities,
      { nameBy: 'tag' },
    );

    expect(result.legendData).toEqual(['worker-1', 'worker-2']);
  });

  it('falls back to entity names for workers without a tag', () => {
    expect(
      mapEntityLines([metric('fn-a', null)], entities, { nameBy: 'tag' })
        .series[0]?.name,
    ).toBe('Send email');
  });
});
