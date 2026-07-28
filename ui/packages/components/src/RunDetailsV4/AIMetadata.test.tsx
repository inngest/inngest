import { describe, expect, it } from 'vitest';

import { traceHasAIMetadata } from './AIMetadata';
import type { SpanMetadata, Trace } from './types';

const makeSpan = (overrides: Partial<Trace> = {}): Trace => ({
  attempts: 0,
  endedAt: '2026-01-01T00:00:05Z',
  isRoot: false,
  name: 'span',
  outputID: null,
  queuedAt: '2026-01-01T00:00:00Z',
  scheduledAt: null,
  spanID: 'span-1',
  startedAt: '2026-01-01T00:00:01Z',
  status: 'COMPLETED',
  stepInfo: null,
  userlandSpan: null,
  isUserland: false,
  ...overrides,
});

const aiMetadata: SpanMetadata = {
  scope: 'extended_trace',
  kind: 'inngest.ai',
  updatedAt: '2026-01-01T00:00:02Z',
  values: { request_model: 'claude-opus-4', input_tokens: 10, output_tokens: 20 },
};

const timingMetadata: SpanMetadata = {
  scope: 'step_attempt',
  kind: 'inngest.timing',
  updatedAt: '2026-01-01T00:00:02Z',
  values: { total_inngest_ms: 12 },
};

describe('traceHasAIMetadata', () => {
  it('finds AI metadata nested under descendant spans', () => {
    const trace = makeSpan({
      isRoot: true,
      metadata: [timingMetadata],
      childrenSpans: [makeSpan({ childrenSpans: [makeSpan({ metadata: [aiMetadata] })] })],
    });

    expect(traceHasAIMetadata(trace)).toBe(true);
  });

  it('reports no AI metadata when only other kinds are present', () => {
    const trace = makeSpan({
      isRoot: true,
      metadata: [timingMetadata],
      childrenSpans: [makeSpan({ metadata: [] })],
    });

    expect(traceHasAIMetadata(trace)).toBe(false);
  });
});
