import { describe, expect, it } from 'vitest';

import {
  attachRun,
  clearThread,
  isThreadWaiting,
  pendingRuns,
  settleTurn,
  startTurn,
  type Threads,
} from './chatMessages';
import type { Message } from './types';

const RUN = { eventId: 'evt_1', receipt: 'sig' };

const userMessage: Message = {
  id: 'user_1',
  role: 'user',
  threadId: 't1',
  parts: [{ type: 'text', content: 'how many runs failed?' }],
};

function sent(): Threads {
  return startTurn({}, 't1', userMessage, 'turn_1');
}

describe('startTurn', () => {
  it('adds the message and a waiting turn', () => {
    const threads = sent();

    expect(threads.t1?.map((m) => m.id)).toEqual(['user_1', 'turn_1']);
    expect(isThreadWaiting(threads, 't1')).toBe(true);
    // Nothing to poll until the send returns an event id.
    expect(pendingRuns(threads)).toEqual([]);
  });
});

describe('attachRun', () => {
  it('makes the turn pollable', () => {
    const threads = attachRun(sent(), 't1', 'turn_1', RUN);

    expect(pendingRuns(threads)).toEqual([
      { threadId: 't1', turnId: 'turn_1', run: RUN },
    ]);
  });

  it('ignores a turn that already settled', () => {
    const settled = settleTurn(sent(), 't1', 'turn_1', [
      { type: 'text', content: 'done' },
    ]);

    expect(attachRun(settled, 't1', 'turn_1', RUN)).toBe(settled);
    expect(pendingRuns(settled)).toEqual([]);
  });
});

describe('settleTurn', () => {
  it('replaces the waiting turn in place', () => {
    const threads = settleTurn(
      attachRun(sent(), 't1', 'turn_1', RUN),
      't1',
      'turn_1',
      [{ type: 'text', content: '12 runs failed' }],
    );

    expect(threads.t1?.map((m) => m.id)).toEqual(['user_1', 'turn_1']);
    expect(threads.t1?.[1]?.parts).toEqual([
      { type: 'text', content: '12 runs failed' },
    ]);
    expect(threads.t1?.[1]?.run).toBeUndefined();
    expect(isThreadWaiting(threads, 't1')).toBe(false);
  });

  it('drops the turn when there is nothing to render', () => {
    const threads = settleTurn(sent(), 't1', 'turn_1', []);

    expect(threads.t1?.map((m) => m.id)).toEqual(['user_1']);
    expect(isThreadWaiting(threads, 't1')).toBe(false);
  });

  it('is a no-op the second time, so a repeated result changes nothing', () => {
    const once = settleTurn(sent(), 't1', 'turn_1', [
      { type: 'text', content: 'first' },
    ]);
    const twice = settleTurn(once, 't1', 'turn_1', [
      { type: 'text', content: 'second' },
    ]);

    expect(twice).toBe(once);
  });

  it('has nowhere to land once the thread is cleared', () => {
    const cleared = clearThread(attachRun(sent(), 't1', 'turn_1', RUN), 't1');

    expect(cleared.t1).toBeUndefined();
    expect(
      settleTurn(cleared, 't1', 'turn_1', [{ type: 'text', content: 'late' }]),
    ).toBe(cleared);
  });
});

describe('pendingRuns', () => {
  it('reports a run per waiting thread', () => {
    const both = attachRun(
      startTurn(
        attachRun(sent(), 't1', 'turn_1', RUN),
        't2',
        userMessage,
        'turn_2',
      ),
      't2',
      'turn_2',
      { eventId: 'evt_2', receipt: 'sig2' },
    );

    expect(pendingRuns(both).map((p) => p.run.eventId)).toEqual([
      'evt_1',
      'evt_2',
    ]);
  });
});
