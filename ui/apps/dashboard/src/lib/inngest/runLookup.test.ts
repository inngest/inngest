import { describe, expect, it, vi } from 'vitest';

import { fetchEventRunResult } from './runLookup';

const args = { eventId: 'evt_1', signingKey: 'signkey' };

const ANSWER = { threadId: 't1', summary: 'done' };

// v2 carries the output, v1 the corrected status. Either can be given as null
// to stand for a lookup that didn't come back.
function fakeFetch(
  v2: { output?: unknown }[] | null,
  v1: { status?: string }[] | null,
) {
  return vi.fn(async (url: string) => {
    const isV2 = String(url).includes('/api/v2/');
    const data = isV2 ? v2 : v1;
    return {
      ok: data !== null,
      json: async () => ({ data }),
    } as Response;
  }) as unknown as typeof fetch;
}

describe('fetchEventRunResult', () => {
  it('reads output from v2 and terminality from v1', async () => {
    const fetchImpl = fakeFetch(
      [{ output: ANSWER }],
      [{ status: 'Completed' }],
    );

    expect(await fetchEventRunResult({ ...args, fetchImpl })).toEqual({
      runs: [{ output: ANSWER }],
      failed: false,
    });

    const urls = (
      fetchImpl as unknown as { mock: { calls: string[][] } }
    ).mock.calls.map((c) => c[0]);
    expect(urls[0]).toContain('/api/v2/events/evt_1/runs?includeOutput=true');
    expect(urls[1]).toContain('/v1/events/evt_1/runs');
  });

  it('passes through the step output of a run still working', async () => {
    const stepOutput = { observation: 'ok' };
    const fetchImpl = fakeFetch(
      [{ output: stepOutput }],
      [{ status: 'Running' }],
    );

    expect(await fetchEventRunResult({ ...args, fetchImpl })).toEqual({
      runs: [{ output: stepOutput }],
      failed: false,
    });
  });

  it('reports a failed run, which v2 alone would still call running', async () => {
    const fetchImpl = fakeFetch(
      [{ output: { error: { message: 'boom' } } }],
      [{ status: 'Failed' }],
    );

    expect(await fetchEventRunResult({ ...args, fetchImpl })).toMatchObject({
      failed: true,
    });
  });

  it('is not failed while one run of the event is still going', async () => {
    const fetchImpl = fakeFetch(
      [],
      [{ status: 'Failed' }, { status: 'Running' }],
    );

    expect(await fetchEventRunResult({ ...args, fetchImpl })).toMatchObject({
      failed: false,
    });
  });

  it('is not failed before the event has any runs', async () => {
    const fetchImpl = fakeFetch([], []);

    expect(await fetchEventRunResult({ ...args, fetchImpl })).toEqual({
      runs: [],
      failed: false,
    });
  });

  it('keeps waiting when a lookup fails', async () => {
    expect(
      await fetchEventRunResult({ ...args, fetchImpl: fakeFetch(null, null) }),
    ).toEqual({ runs: [], failed: false });
  });

  it('does nothing without a signing key', async () => {
    const fetchImpl = fakeFetch(
      [{ output: ANSWER }],
      [{ status: 'Completed' }],
    );

    expect(
      await fetchEventRunResult({
        eventId: 'evt_1',
        signingKey: undefined,
        fetchImpl,
      }),
    ).toEqual({ runs: [], failed: false });
    expect(fetchImpl).not.toHaveBeenCalled();
  });
});
