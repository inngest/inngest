import { describe, expect, it } from 'vitest';

import { isScoreMetadata, isWarningMetadata, type SpanMetadata } from './types';

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

  it('matches per name inngest.score.<name> kinds', () => {
    const md = {
      scope: 'step',
      kind: 'inngest.score.latency_ms',
      updatedAt: '2026-06-23T00:00:00Z',
      values: { value: 1 },
    } as unknown as SpanMetadata;

    expect(isScoreMetadata(md)).toBe(true);
  });

  it('does not match a score prefix w/o a name', () => {
    const md = {
      scope: 'step',
      kind: 'inngest.score.',
      updatedAt: '2026-06-23T00:00:00Z',
      values: { value: 1 },
    } as unknown as SpanMetadata;

    expect(isScoreMetadata(md)).toBe(false);
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

describe('isWarningMetadata', () => {
  it.each([
    ['inngest.warnings', { 'sdk.size': 'too big' }],
    ['inngest.warning.sdk.size', { 'sdk.size': 'too big' }],
  ])('matches %s', (kind, values) => {
    const md = { scope: 'run', kind, updatedAt: 't', values } as unknown as SpanMetadata;

    expect(isWarningMetadata(md)).toBe(true);
  });

  it.each(['inngest.warning.', 'inngest.warning', 'inngest.score', 'userland.warnings'])(
    'does not match %s',
    (kind) => {
      const md = { scope: 'run', kind, updatedAt: 't', values: {} } as unknown as SpanMetadata;

      expect(isWarningMetadata(md)).toBe(false);
    }
  );
});
