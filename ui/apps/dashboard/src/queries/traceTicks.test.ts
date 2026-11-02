import { describe, expect, it } from 'vitest';

import { indexTicks, mergeTicks, type SpanTick } from './traceTicks';

const tick: SpanTick = {
  id: 'message-1',
  sources: ['pause-1'],
  at: '2026-11-01T10:00:00Z',
  readAt: null,
};

describe('indexTicks', () => {
  it('finds ticks at every depth and skips spans without any', () => {
    const bySpanID = indexTicks({
      spanID: 'root',
      ticks: [],
      childrenSpans: [
        {
          spanID: 'inbox',
          ticks: [tick],
          childrenSpans: [
            { spanID: 'deep', ticks: [{ ...tick, id: 'message-2' }] },
          ],
        },
        { spanID: 'plain', ticks: [] },
      ],
    });

    expect([...bySpanID.keys()].sort()).toEqual(['deep', 'inbox']);
    expect(bySpanID.get('inbox')).toEqual([tick]);
    expect(bySpanID.get('deep')?.[0]?.id).toBe('message-2');
  });

  it('is empty when the ticks query returned no trace', () => {
    expect(indexTicks(null).size).toBe(0);
    expect(indexTicks(undefined).size).toBe(0);
  });
});

describe('mergeTicks', () => {
  it('puts each span its own ticks and leaves the rest of the span alone', () => {
    const trace = {
      spanID: 'root',
      name: 'run',
      childrenSpans: [
        { spanID: 'inbox', name: 'inbox' },
        { spanID: 'plain', name: 'plain' },
      ],
    };

    const merged = mergeTicks(trace, new Map([['inbox', [tick]]]));

    expect(merged.name).toBe('run');
    expect(merged.ticks).toEqual([]);
    expect(merged.childrenSpans).toEqual([
      { spanID: 'inbox', name: 'inbox', ticks: [tick] },
      { spanID: 'plain', name: 'plain', ticks: [] },
    ]);
  });

  it('does not change the trace it was given', () => {
    const trace = { spanID: 'root', childrenSpans: [{ spanID: 'inbox' }] };

    mergeTicks(trace, new Map([['inbox', [tick]]]));

    expect(trace).toEqual({
      spanID: 'root',
      childrenSpans: [{ spanID: 'inbox' }],
    });
  });

  it('gives a span without children no childrenSpans', () => {
    const merged = mergeTicks({ spanID: 'root' }, new Map());

    expect(merged).toEqual({ spanID: 'root', ticks: [] });
  });
});
