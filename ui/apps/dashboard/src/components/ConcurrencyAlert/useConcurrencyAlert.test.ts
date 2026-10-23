// @vitest-environment jsdom
import { cleanup, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  ConcurrencyBurstUsageDocument,
  useConcurrencyAlert,
  type ConcurrencyAlertCheckQuery,
} from './useConcurrencyAlert';

const mocks = vi.hoisted(() => ({ useQuery: vi.fn() }));

vi.mock('urql', async (importOriginal) => ({
  ...(await importOriginal<typeof import('urql')>()),
  useQuery: mocks.useQuery,
}));

function account({
  bursting,
  limitReached,
}: {
  bursting: boolean;
  limitReached: boolean;
}) {
  return {
    account: {
      id: 'acct',
      burstConcurrency: {
        limit: 300,
        budgetMinutes: 100,
        bursting,
      },
      concurrencyStatus: {
        limitReached,
        minutesWithHits: limitReached ? 3 : 0,
      },
      entitlements: { concurrency: { limit: 100 } },
    },
  };
}

const usage = {
  account: { burstConcurrency: { usage: { minutesRemaining: 40 } } },
};

function mockQueries(data: ConcurrencyAlertCheckQuery | undefined) {
  mocks.useQuery.mockImplementation(({ query }) =>
    query === ConcurrencyBurstUsageDocument
      ? [{ data: data?.account.burstConcurrency?.bursting ? usage : undefined }]
      : [{ data }],
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('useConcurrencyAlert', () => {
  it('reports boosted with the burst limit while bursting', () => {
    mockQueries(account({ bursting: true, limitReached: true }));

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current?.state).toBe('boosted');
    expect(result.current?.limit).toBe(300);
    expect(result.current?.usedRatio).toBe(0.6);
  });

  it('reports limited with the base limit when not bursting', () => {
    mockQueries(account({ bursting: false, limitReached: true }));

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current?.state).toBe('limited');
    expect(result.current?.limit).toBe(100);
    expect(mocks.useQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        query: ConcurrencyBurstUsageDocument,
        pause: true,
      }),
    );
  });

  it('reports nothing when neither bursting nor limited', () => {
    mockQueries(account({ bursting: false, limitReached: false }));

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current).toBeNull();
  });

  it('reports limited when burst is off for the account', () => {
    const data = account({ bursting: false, limitReached: true });
    mockQueries({ account: { ...data.account, burstConcurrency: null } });

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current?.state).toBe('limited');
    expect(result.current?.minutesRemaining).toBeUndefined();
    expect(result.current?.usedRatio).toBe(0);
  });

  it('reports nothing while the query is loading', () => {
    mockQueries(undefined);

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current).toBeNull();
  });
});
