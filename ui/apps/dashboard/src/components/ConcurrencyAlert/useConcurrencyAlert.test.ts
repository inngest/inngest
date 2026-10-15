// @vitest-environment jsdom
import { cleanup, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useConcurrencyAlert } from './useConcurrencyAlert';

const mocks = vi.hoisted(() => ({
  useQuery: vi.fn(),
  useAccountConcurrencyPressure: vi.fn(),
}));

vi.mock('urql', async (importOriginal) => ({
  ...(await importOriginal<typeof import('urql')>()),
  useQuery: mocks.useQuery,
}));

vi.mock('../Runs/useAccountConcurrencyPressure', () => ({
  useAccountConcurrencyPressure: mocks.useAccountConcurrencyPressure,
}));

const now = Date.UTC(2026, 9, 15, 12, 0, 30);

function account({ burstEndedAt }: { burstEndedAt: number | null }) {
  return {
    account: {
      marketplaceBillingURL: null,
      burstConcurrency: {
        limit: 300,
        budgetMinutes: 100,
        exhaustedAt: null,
        usage: {
          minutesRemaining: 40,
          bursts:
            burstEndedAt == null
              ? []
              : [{ endedAt: new Date(burstEndedAt).toISOString() }],
        },
      },
      entitlements: { concurrency: { limit: 100 } },
    },
  };
}

function pressure(isPressured: boolean) {
  return {
    isPressured,
    minutesWithHits: 0,
    accountID: 'acct',
    isMarketplace: false,
    isFreePlan: false,
  };
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] });
  vi.setSystemTime(now);
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.clearAllMocks();
});

describe('useConcurrencyAlert', () => {
  it('reports boosted with the burst limit and skips the pressure query', () => {
    mocks.useQuery.mockReturnValue([
      { data: account({ burstEndedAt: now }), fetching: false },
    ]);
    mocks.useAccountConcurrencyPressure.mockReturnValue(pressure(true));

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current?.state).toBe('boosted');
    expect(result.current?.limit).toBe(300);
    expect(result.current?.usedRatio).toBe(0.6);
    expect(mocks.useAccountConcurrencyPressure).toHaveBeenLastCalledWith({
      skip: true,
    });
  });

  it('reports limited with the base limit when not boosted', () => {
    mocks.useQuery.mockReturnValue([
      { data: account({ burstEndedAt: null }), fetching: false },
    ]);
    mocks.useAccountConcurrencyPressure.mockReturnValue(pressure(true));

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current?.state).toBe('limited');
    expect(result.current?.limit).toBe(100);
    expect(mocks.useAccountConcurrencyPressure).toHaveBeenLastCalledWith({
      skip: false,
    });
  });

  it('holds the pressure query until the burst query settles', () => {
    mocks.useQuery.mockReturnValue([{ data: undefined, fetching: true }]);
    mocks.useAccountConcurrencyPressure.mockReturnValue(pressure(false));

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current).toBeNull();
    expect(mocks.useAccountConcurrencyPressure).toHaveBeenLastCalledWith({
      skip: true,
    });
  });

  it('checks pressure when the burst query fails', () => {
    mocks.useQuery.mockReturnValue([
      { data: undefined, error: new Error('boom'), fetching: false },
    ]);
    mocks.useAccountConcurrencyPressure.mockReturnValue(pressure(true));

    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current?.state).toBe('limited');
    expect(result.current?.limit).toBeUndefined();
  });

  it('judges a shared result against the time it first arrived', () => {
    const data = account({ burstEndedAt: now });
    mocks.useQuery.mockReturnValue([{ data, fetching: false }]);
    mocks.useAccountConcurrencyPressure.mockReturnValue(pressure(false));

    renderHook(() => useConcurrencyAlert());
    vi.setSystemTime(now + 10 * 60_000);
    const { result } = renderHook(() => useConcurrencyAlert());

    expect(result.current?.state).toBe('boosted');
  });
});
