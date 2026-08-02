import { afterEach, describe, expect, it, vi } from 'vitest';

import { fetchRunResult } from './useInsightsAgent';

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
  it('returns the run whose output matches the thread', async () => {
    mockResponse({
      runs: [
        { status: 'Completed', output: { threadId: 'other', summary: 'no' } },
        { status: 'Completed', output: { threadId: 'mine', summary: 'yes' } },
      ],
    });

    const result = await fetchRunResult('evt_1', 'mine');
    expect(result?.output).toMatchObject({ summary: 'yes' });
  });

  it('falls back to the only run when no output carries a thread id', async () => {
    mockResponse({ runs: [{ status: 'Running', output: null }] });

    const result = await fetchRunResult('evt_1', 'mine');
    expect(result?.status).toBe('Running');
  });

  it('returns null when the lookup finds nothing', async () => {
    mockResponse({ runs: [] });
    expect(await fetchRunResult('evt_1', 'mine')).toBeNull();
  });

  it('returns null on a non-ok response', async () => {
    mockResponse({}, false);
    expect(await fetchRunResult('evt_1', 'mine')).toBeNull();
  });
});
