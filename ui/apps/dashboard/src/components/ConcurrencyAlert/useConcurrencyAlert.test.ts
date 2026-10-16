// @vitest-environment jsdom
import { cleanup, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useConcurrencyAlert } from './useConcurrencyAlert';

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
      marketplace: null,
      marketplaceBillingURL: null,
      plan: { isFree: false },
      burstConcurrency: {
        limit: 300,
        budgetMinutes: 100,
        usage: { minutesRemaining: 40, bursting },
      },
      concurrencyStatus: { limitReached },
      entitlements: { concurrency: { limit: 100 } },
    },
  };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('useConcurrencyAlert', () => {
  it('reports boosted with the burst limit while bursting', () => {
    mocks.useQuery.mockReturnValue([
      { data: account({ bursting: true, limitReached: true }) },
    ]);

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current?.state).toBe('boosted');
    expect(result.current?.limit).toBe(300);
    expect(result.current?.usedRatio).toBe(0.6);
  });

  it('reports limited with the base limit when not bursting', () => {
    mocks.useQuery.mockReturnValue([
      { data: account({ bursting: false, limitReached: true }) },
    ]);

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current?.state).toBe('limited');
    expect(result.current?.limit).toBe(100);
  });

  it('reports nothing when neither bursting nor limited', () => {
    mocks.useQuery.mockReturnValue([
      { data: account({ bursting: false, limitReached: false }) },
    ]);

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current).toBeNull();
  });

  it('reports limited when burst is off for the account', () => {
    const data = account({ bursting: false, limitReached: true });
    mocks.useQuery.mockReturnValue([
      { data: { account: { ...data.account, burstConcurrency: null } } },
    ]);

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current?.state).toBe('limited');
    expect(result.current?.minutesRemaining).toBeUndefined();
    expect(result.current?.usedRatio).toBe(0);
  });

  it('reports nothing while the query is loading', () => {
    mocks.useQuery.mockReturnValue([{ data: undefined }]);

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current).toBeNull();
  });
});
