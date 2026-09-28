// @vitest-environment jsdom
import type { PropsWithChildren } from 'react';
import { act } from 'react-dom/test-utils';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import {
  ClientFeatureFlagProvider,
  IDENTIFICATION_TIMEOUT_MS,
} from './ClientFeatureFlagProvider';
import { useBooleanFlag } from './hooks';

const roots: Array<{ root: Root; container: HTMLDivElement }> = [];
const cleanup = () => {
  for (const { root, container } of roots.splice(0)) {
    act(() => root.unmount());
    container.remove();
  }
};
const renderHook = <T,>(
  hook: () => T,
  { wrapper: Wrapper }: { wrapper: React.ComponentType<PropsWithChildren> },
) => {
  const container = document.createElement('div');
  document.body.appendChild(container);
  const root = createRoot(container);
  roots.push({ root, container });
  const result = {} as { current: T };
  const Probe = () => {
    result.current = hook();
    return null;
  };
  const rerender = () =>
    act(() =>
      root.render(
        <Wrapper>
          <Probe />
        </Wrapper>,
      ),
    );
  rerender();
  return { result, rerender };
};
const waitFor = async (assert: () => void) => {
  await act(async () => {});
  assert();
};
(
  globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  flags: {} as Record<string, unknown>,
  identify: vi.fn(),
  error: undefined as Error | undefined,
  accountID: 'account-1',
  hasClerkIdentity: true,
}));
vi.mock('launchdarkly-react-client-sdk', () => ({
  useFlags: () => mocks.flags,
  useLDClient: () => client,
  useLDClientError: () => mocks.error,
  withLDProvider: () => (component: unknown) => component,
}));
const client = { identify: mocks.identify };
const flag = 'polling-disabled';
vi.mock('@clerk/tanstack-react-start', () => ({
  useUser: () => ({
    isLoaded: true,
    user: mocks.hasClerkIdentity
      ? { externalId: 'user-1', fullName: 'User' }
      : null,
  }),
  useOrganization: () => ({
    isLoaded: true,
    organization: mocks.hasClerkIdentity
      ? {
          name: 'Account',
          publicMetadata: { accountID: mocks.accountID },
        }
      : null,
  }),
}));
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  mocks.flags = {};
  mocks.identify.mockReset();
  mocks.error = undefined;
  mocks.accountID = 'account-1';
  mocks.hasClerkIdentity = true;
  vi.useRealTimers();
});
it.each([undefined, 'true', null, 1])(
  'uses a ready default for invalid flag %s and recovers',
  async (value) => {
    mocks.identify.mockResolvedValue({});
    mocks.flags = { [flag]: value };
    const log = vi.spyOn(console, 'error').mockImplementation(() => {});
    const { result, rerender } = renderHook(() => useBooleanFlag(flag, false), {
      wrapper: ClientFeatureFlagProvider,
    });
    await waitFor(() =>
      expect(result.current).toEqual({ isReady: true, value: false }),
    );
    expect(log).toHaveBeenCalledWith(
      expect.stringContaining('using default false'),
    );
    const calls = log.mock.calls.length;
    rerender();
    expect(log).toHaveBeenCalledTimes(calls);
    mocks.flags = { [flag]: true };
    rerender();
    expect(result.current).toEqual({ isReady: true, value: true });
  },
);
it('keeps pending identification unready and uses defaults after rejection', async () => {
  let reject!: (error: Error) => void;
  mocks.identify.mockReturnValue(
    new Promise((_, fail) => {
      reject = fail;
    }),
  );
  mocks.flags = { [flag]: false };
  vi.spyOn(console, 'error').mockImplementation(() => {});
  const { result, rerender } = renderHook(() => useBooleanFlag(flag, true), {
    wrapper: ClientFeatureFlagProvider,
  });
  expect(result.current).toEqual({ isReady: false, value: true });
  await act(async () => reject(new Error('offline')));
  expect(result.current).toEqual({ isReady: true, value: true });
  mocks.accountID = 'account-2';
  mocks.identify.mockResolvedValue({});
  rerender();
  expect(result.current).toEqual({ isReady: false, value: true });
  await waitFor(() =>
    expect(result.current).toEqual({ isReady: true, value: false }),
  );
});
it('uses defaults on client initialization errors', () => {
  mocks.error = new Error('initialization failed');
  vi.spyOn(console, 'error').mockImplementation(() => {});
  const { result } = renderHook(() => useBooleanFlag(flag, true), {
    wrapper: ClientFeatureFlagProvider,
  });
  expect(result.current).toEqual({ isReady: true, value: true });
});
it('uses ready defaults when JWT auth has no Clerk identity', () => {
  mocks.hasClerkIdentity = false;
  mocks.flags = { [flag]: true };

  const { result } = renderHook(() => useBooleanFlag(flag), {
    wrapper: ClientFeatureFlagProvider,
  });

  expect(result.current).toEqual({ isReady: true, value: false });
  expect(mocks.identify).not.toHaveBeenCalled();
});
it('uses ready defaults when identification times out', async () => {
  vi.useFakeTimers();
  let resolve!: () => void;
  mocks.flags = { [flag]: false };
  mocks.identify.mockReturnValue(
    new Promise<void>((done) => {
      resolve = done;
    }),
  );
  vi.spyOn(console, 'error').mockImplementation(() => {});

  const { result } = renderHook(() => useBooleanFlag(flag, true), {
    wrapper: ClientFeatureFlagProvider,
  });
  expect(result.current).toEqual({ isReady: false, value: true });

  await act(async () => {
    await vi.advanceTimersByTimeAsync(IDENTIFICATION_TIMEOUT_MS);
  });

  expect(result.current).toEqual({ isReady: true, value: true });
  await act(async () => resolve());
  expect(result.current).toEqual({ isReady: true, value: true });
});
it('ignores identification results from an old account', async () => {
  let resolve!: () => void;
  mocks.identify.mockReturnValueOnce(
    new Promise<void>((done) => {
      resolve = done;
    }),
  );
  mocks.identify.mockReturnValueOnce(new Promise(() => {}));
  const { result, rerender } = renderHook(() => useBooleanFlag(flag), {
    wrapper: ClientFeatureFlagProvider,
  });
  await act(async () => {});
  mocks.accountID = 'account-2';
  rerender();
  await act(async () => resolve());
  expect(result.current.isReady).toBe(false);
});
