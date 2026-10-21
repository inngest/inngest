// @vitest-environment jsdom
import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  current: {} as Record<string, string>,
  getSignupMetadata: vi.fn(),
}));

vi.mock('./signupAttribution', () => ({
  getSignupMetadata: mocks.getSignupMetadata,
}));

import { useSignupMetadata } from './useSignupMetadata';

const gaIds = { gaClientId: '123.456', gaSessionId: '789' };

const advance = (ms: number) => {
  act(() => {
    vi.advanceTimersByTime(ms);
  });
};

beforeEach(() => {
  vi.useFakeTimers();
  mocks.current = {};
  mocks.getSignupMetadata.mockImplementation(() => ({ ...mocks.current }));
});

afterEach(() => {
  vi.useRealTimers();
  vi.clearAllMocks();
});

describe('useSignupMetadata', () => {
  it('picks up cookies that appear after mount', () => {
    const { result } = renderHook(() => useSignupMetadata());
    expect(result.current).toEqual({});

    mocks.current = { ...gaIds };
    advance(1000);

    expect(result.current).toEqual(gaIds);
  });

  it('keeps polling after the GA IDs appear until a click ID shows up', () => {
    const { result } = renderHook(() => useSignupMetadata());

    mocks.current = { ...gaIds };
    advance(1000);
    const callsWithGaOnly = mocks.getSignupMetadata.mock.calls.length;

    // GA IDs alone must not stop the polling.
    advance(3000);
    expect(mocks.getSignupMetadata.mock.calls.length).toBe(callsWithGaOnly + 3);

    // The Conversion Linker writes its cookie later.
    mocks.current = { ...gaIds, gclid: 'late_click_id' };
    advance(1000);
    expect(result.current).toEqual({ ...gaIds, gclid: 'late_click_id' });

    // Everything is present, so polling stops.
    const callsWhenComplete = mocks.getSignupMetadata.mock.calls.length;
    advance(5000);
    expect(mocks.getSignupMetadata.mock.calls.length).toBe(callsWhenComplete);
  });

  it('stops immediately when everything is already present at mount', () => {
    mocks.current = { ...gaIds, gbraid: 'wbraid_or_gbraid' };
    renderHook(() => useSignupMetadata());
    const calls = mocks.getSignupMetadata.mock.calls.length;

    advance(5000);

    expect(mocks.getSignupMetadata.mock.calls.length).toBe(calls);
  });

  it('gives up after a bounded number of refreshes when there is no click ID', () => {
    mocks.current = { ...gaIds };
    renderHook(() => useSignupMetadata());

    advance(30_000);
    const calls = mocks.getSignupMetadata.mock.calls.length;
    advance(30_000);

    expect(mocks.getSignupMetadata.mock.calls.length).toBe(calls);
    // Initial read, the effect's first read, and 20 refreshes.
    expect(calls).toBe(22);
  });

  it('stops polling on unmount', () => {
    const { unmount } = renderHook(() => useSignupMetadata());
    advance(2000);
    unmount();
    const calls = mocks.getSignupMetadata.mock.calls.length;

    advance(5000);

    expect(mocks.getSignupMetadata.mock.calls.length).toBe(calls);
  });
});
