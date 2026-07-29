import { describe, expect, it } from 'vitest';

import { collectScoreMetadata, scoreRows } from './ScoresAttrs';

describe('scoreRows', () => {
  it('produces one row per score name from the values map, sorted by name', () => {
    const rows = scoreRows([
      {
        kind: 'inngest.score',
        updatedAt: '2026-06-23T00:00:00Z',
        values: { query_writer_latency_ms: { value: 3255 }, accuracy: { value: true } },
      },
    ]);

    expect(rows).toEqual([
      { name: 'accuracy', value: true, updatedAt: '2026-06-23T00:00:00Z' },
      { name: 'query_writer_latency_ms', value: 3255, updatedAt: '2026-06-23T00:00:00Z' },
    ]);
  });

  it('drops entries whose value is not a finite number or boolean', () => {
    const rows = scoreRows([
      {
        kind: 'inngest.score',
        updatedAt: 't',
        values: { ok: { value: 1 }, nan: { value: NaN }, str: { value: 'x' } },
      },
    ]);

    expect(rows.map((r) => r.name)).toEqual(['ok']);
  });
});

describe('collectScoreMetadata', () => {
  it('collects inngest.score metadata from the span and nested children', () => {
    const trace = {
      metadata: [{ kind: 'inngest.score', updatedAt: 't', values: { a: { value: 1 } } }],
      childrenSpans: [
        { metadata: [{ kind: 'inngest.score', updatedAt: 't', values: { b: { value: 2 } } }] },
        { metadata: [{ kind: 'inngest.experiment', updatedAt: 't', values: {} }] },
      ],
    };

    expect(collectScoreMetadata(trace)).toHaveLength(2);
  });
});
