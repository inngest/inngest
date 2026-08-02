import { describe, expect, it, vi } from 'vitest';

import { fetchRunsForEvent } from './runLookup';

const RUNS = [{ status: 'Completed', output: { summary: 'secret' } }];

// Responds to /v1/events/{id} with the given owner, and to the runs path with RUNS.
function fakeFetch(ownerUserId: string | undefined, runs = RUNS) {
  return vi.fn(async (url: string) => {
    const body = String(url).endsWith('/runs')
      ? { data: runs }
      : { data: { data: { userId: ownerUserId } } };
    return { ok: true, json: async () => body } as Response;
  }) as unknown as typeof fetch;
}

const args = { eventId: 'evt_1', signingKey: 'signkey' };

describe('fetchRunsForEvent', () => {
  it('returns runs to the user who triggered the event', async () => {
    const result = await fetchRunsForEvent({
      ...args,
      userId: 'user_a',
      fetchImpl: fakeFetch('user_a'),
    });

    expect(result).toEqual([{ status: 'Completed', output: RUNS[0].output }]);
  });

  it('returns nothing to a different user', async () => {
    const fetchImpl = fakeFetch('user_a');
    const result = await fetchRunsForEvent({
      ...args,
      userId: 'user_b',
      fetchImpl,
    });

    expect(result).toEqual([]);
    // Bailed before ever asking for the output.
    expect(fetchImpl).toHaveBeenCalledTimes(1);
  });

  it('returns nothing when the event has no owner recorded', async () => {
    const result = await fetchRunsForEvent({
      ...args,
      userId: 'user_a',
      fetchImpl: fakeFetch(undefined),
    });

    expect(result).toEqual([]);
  });

  it('returns nothing when the event lookup fails', async () => {
    const fetchImpl = vi.fn(async () => ({
      ok: false,
      json: async () => ({}),
    })) as unknown as typeof fetch;

    expect(
      await fetchRunsForEvent({ ...args, userId: 'user_a', fetchImpl }),
    ).toEqual([]);
  });

  it('returns nothing without a signing key', async () => {
    const fetchImpl = fakeFetch('user_a');

    expect(
      await fetchRunsForEvent({
        eventId: 'evt_1',
        userId: 'user_a',
        signingKey: undefined,
        fetchImpl,
      }),
    ).toEqual([]);
    expect(fetchImpl).not.toHaveBeenCalled();
  });
});
