import { afterEach, describe, expect, it, vi } from 'vitest';

import { fetchRunResult } from './useInsightsAgent';

const RUN = { eventId: 'evt_1', receipt: 'sig' };

function mockResponse(body: unknown, ok = true) {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok,
      json: async () => body,
    }),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('fetchRunResult', () => {
  it('returns the answer for this thread', async () => {
    mockResponse({
      runs: [
        { output: { threadId: 'other', summary: 'no' } },
        { output: { threadId: 'mine', summary: 'yes' } },
      ],
      failed: false,
    });

    const result = await fetchRunResult(RUN, 'mine');

    expect(result?.answer).toMatchObject({ summary: 'yes' });
  });

  it('ignores a step output, which is what an in-flight run reports', async () => {
    // The last step to finish, not the run's answer: no thread id.
    mockResponse({
      runs: [
        { output: { observation: 'ok', draftPatch: { sql: 'SELECT 1' } } },
      ],
      failed: false,
    });

    expect(await fetchRunResult(RUN, 'mine')).toEqual({
      answer: null,
      failed: false,
    });
  });

  it('reports a failure with no answer to show for it', async () => {
    mockResponse({ runs: [{ output: null }], failed: true });

    expect(await fetchRunResult(RUN, 'mine')).toEqual({
      answer: null,
      failed: true,
    });
  });

  it('reports nothing yet before the event has runs', async () => {
    mockResponse({ runs: [], failed: false });

    expect(await fetchRunResult(RUN, 'mine')).toEqual({
      answer: null,
      failed: false,
    });
  });

  it('returns null on a non-ok response', async () => {
    mockResponse({}, false);
    expect(await fetchRunResult(RUN, 'mine')).toBeNull();
  });
});
