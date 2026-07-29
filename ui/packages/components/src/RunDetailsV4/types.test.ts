import { describe, expect, it } from 'vitest';

import { isScoreMetadata, type SpanMetadata } from './types';

describe('isScoreMetadata', () => {
  it('matches the constant inngest.score kind', () => {
    const md = {
      scope: 'step',
      kind: 'inngest.score',
      updatedAt: '2026-06-23T00:00:00Z',
      values: { latency_ms: { value: 1 } },
    } as unknown as SpanMetadata;

    expect(isScoreMetadata(md)).toBe(true);
  });

  it('does not match non-score kinds', () => {
    const md = {
      scope: 'step',
      kind: 'inngest.experiment',
      updatedAt: '2026-06-23T00:00:00Z',
      values: {},
    } as unknown as SpanMetadata;

    expect(isScoreMetadata(md)).toBe(false);
  });
});
