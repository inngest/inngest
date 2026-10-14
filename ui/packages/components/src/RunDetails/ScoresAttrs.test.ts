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

  it('keeps only the latest write when a score name is recorded more than once', () => {
    const rows = scoreRows([
      {
        kind: 'inngest.score',
        updatedAt: '2026-06-23T00:00:09Z',
        values: { relevance: { value: 0.9 } },
      },
      {
        kind: 'inngest.score',
        updatedAt: '2026-06-23T00:00:05Z',
        values: { relevance: { value: 0.5 } },
      },
    ]);

    expect(rows).toEqual([{ name: 'relevance', value: 0.9, updatedAt: '2026-06-23T00:00:09Z' }]);
  });

  it('reads per name score kinds, w/ the name taken from the kind', () => {
    const rows = scoreRows([
      { kind: 'inngest.score.accuracy', updatedAt: 't', values: { value: 0.95 } },
      { kind: 'inngest.score.latency.p99', updatedAt: 't', values: { value: 120 } },
    ]);

    expect(rows).toEqual([
      { name: 'accuracy', value: 0.95, updatedAt: 't' },
      { name: 'latency.p99', value: 120, updatedAt: 't' },
    ]);
  });

  it('drops per name score kinds w/o a valid value', () => {
    const rows = scoreRows([
      { kind: 'inngest.score.ok', updatedAt: 't', values: { value: true } },
      { kind: 'inngest.score.str', updatedAt: 't', values: { value: 'x' } },
      { kind: 'inngest.score.nested', updatedAt: 't', values: { nested: { value: 1 } } },
      { kind: 'inngest.score.', updatedAt: 't', values: { value: 1 } },
    ]);

    expect(rows.map((r) => r.name)).toEqual(['ok']);
  });

  it('merges all score shapes, keeping the latest write per name', () => {
    const trace = {
      metadata: [
        // pre #4482: per name kind
        {
          kind: 'inngest.score.accuracy',
          updatedAt: '2026-06-23T00:00:01Z',
          values: { value: 0.5 },
        },
        // legacy shared kind
        {
          kind: 'inngest.score',
          updatedAt: '2026-06-23T00:00:02Z',
          values: { accuracy: { value: 0.7 }, relevance: { value: 0.8 } },
        },
      ],
      childrenSpans: [
        {
          metadata: [
            // per name kind going forward
            {
              kind: 'inngest.score.relevance',
              updatedAt: '2026-06-23T00:00:03Z',
              values: { value: 0.9 },
            },
            {
              kind: 'inngest.score.passed',
              updatedAt: '2026-06-23T00:00:03Z',
              values: { value: true },
            },
          ],
        },
      ],
    };

    expect(scoreRows(collectScoreMetadata(trace))).toEqual([
      { name: 'accuracy', value: 0.7, updatedAt: '2026-06-23T00:00:02Z' },
      { name: 'passed', value: true, updatedAt: '2026-06-23T00:00:03Z' },
      { name: 'relevance', value: 0.9, updatedAt: '2026-06-23T00:00:03Z' },
    ]);
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

  it('collects per name score kinds and skips look alike kinds', () => {
    const trace = {
      metadata: [
        { kind: 'inngest.score.a', updatedAt: 't', values: { value: 1 } },
        { kind: 'inngest.score.', updatedAt: 't', values: { value: 1 } },
        { kind: 'inngest.scores', updatedAt: 't', values: {} },
        { kind: 'userland.score.a', updatedAt: 't', values: { value: 1 } },
      ],
    };

    expect(collectScoreMetadata(trace).map((md) => md.kind)).toEqual(['inngest.score.a']);
  });
});
