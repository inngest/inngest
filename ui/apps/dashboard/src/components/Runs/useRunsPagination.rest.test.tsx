// @vitest-environment jsdom

import { act } from 'react-dom/test-utils';
import { createRoot, type Root } from 'react-dom/client';
import {
  focusManager,
  onlineManager,
  QueryClient,
  QueryClientProvider,
} from '@tanstack/react-query';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useRunsPagination } from './useRunsPagination';

const mocks = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}));

vi.mock('@/queries/useInngestAPIFetch', () => ({
  useInngestAPIFetch: () => mocks.apiFetch,
}));

(
  globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const commonQueryVars = {
  appIDs: null,
  restAppIDs: null,
  environmentID: 'environment-id',
  functionSlug: null,
  startTime: '2026-09-15T00:00:00Z',
  endTime: null,
  status: null,
  timeField: 'QUEUED_AT',
  celQuery: undefined,
  isDeferred: null,
  environmentSlug: 'production',
  functionAppID: null,
};

type RunsPaginationResult = ReturnType<typeof useRunsPagination>;

function RunsPaginationHarness({
  onRender,
  pause = false,
  celQuery,
}: {
  onRender: (result: RunsPaginationResult) => void;
  pause?: boolean;
  celQuery?: string;
}) {
  onRender(
    useRunsPagination({
      commonQueryVars: { ...commonQueryVars, celQuery },
      pause,
    }),
  );
  return null;
}

function run(id: string) {
  return {
    id,
    function: { id: 'fn-id', name: 'Function', slug: 'function' },
    app: { id: 'app' },
    status: 'COMPLETED',
    queuedAt: '2026-09-15T00:00:00Z',
    startedAt: '2026-09-15T00:00:01Z',
    endedAt: '2026-09-15T00:00:02Z',
  };
}

describe('REST runs pagination refresh', () => {
  let root: Root | undefined;
  let container: HTMLDivElement | undefined;
  let queryClient: QueryClient | undefined;

  afterEach(async () => {
    if (root) {
      await act(async () => root?.unmount());
    }
    queryClient?.clear();
    focusManager.setFocused(undefined);
    onlineManager.setOnline(true);
    container?.remove();
    root = undefined;
    container = undefined;
    queryClient = undefined;
    vi.clearAllMocks();
  });

  it('only refetches manually and refreshes from the first page', async () => {
    mocks.apiFetch.mockImplementation((pathname: string) => {
      const isSecondPage = pathname.includes('cursor=second-page');
      return Promise.resolve(
        new Response(
          JSON.stringify(
            isSecondPage
              ? {
                  data: [run('run-2')],
                  page: { hasMore: false, limit: 40 },
                }
              : {
                  data: [run('run-1')],
                  page: {
                    cursor: 'second-page',
                    hasMore: true,
                    limit: 40,
                  },
                },
          ),
          { status: 200 },
        ),
      );
    });

    let result: RunsPaginationResult | undefined;
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);

    await act(async () => {
      root?.render(
        <QueryClientProvider client={queryClient!}>
          <RunsPaginationHarness onRender={(value) => (result = value)} />
        </QueryClientProvider>,
      );
    });
    await vi.waitFor(() => expect(mocks.apiFetch).toHaveBeenCalledTimes(1));

    await act(async () => {
      focusManager.setFocused(false);
      focusManager.setFocused(true);
      onlineManager.setOnline(false);
      onlineManager.setOnline(true);
    });
    expect(mocks.apiFetch).toHaveBeenCalledTimes(1);

    await act(async () => result?.loadMore());
    await vi.waitFor(() => expect(mocks.apiFetch).toHaveBeenCalledTimes(2));
    await vi.waitFor(() =>
      expect(result?.runs.map(({ id }) => id)).toEqual(['run-1', 'run-2']),
    );

    await act(async () => result?.reset());
    await vi.waitFor(() => expect(mocks.apiFetch).toHaveBeenCalledTimes(3));
    await vi.waitFor(() =>
      expect(result?.runs.map(({ id }) => id)).toEqual(['run-1']),
    );
    expect(mocks.apiFetch.mock.calls[2]?.[0]).not.toContain('cursor=');
  });

  it('does not start REST while metadata loading is paused', async () => {
    let result: RunsPaginationResult | undefined;
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);

    await act(async () => {
      root?.render(
        <QueryClientProvider client={queryClient!}>
          <RunsPaginationHarness pause onRender={(value) => (result = value)} />
        </QueryClientProvider>,
      );
    });

    expect(mocks.apiFetch).not.toHaveBeenCalled();
    expect(result?.isLoadingInitial).toBe(true);
  });

  it('distinguishes the pending scan from a completed zero-match response', async () => {
    let resolveResponse: ((response: Response) => void) | undefined;
    mocks.apiFetch.mockReturnValue(
      new Promise<Response>((resolve) => {
        resolveResponse = resolve;
      }),
    );

    let result: RunsPaginationResult | undefined;
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);

    await act(async () => {
      root?.render(
        <QueryClientProvider client={queryClient!}>
          <RunsPaginationHarness
            celQuery="event.data.userId == 'missing'"
            onRender={(value) => (result = value)}
          />
        </QueryClientProvider>,
      );
    });

    expect(result?.progressiveSearch).toMatchObject({
      phase: 'searching',
      hasCompletedScanResponse: false,
    });
    expect(result?.runs).toEqual([]);

    await act(async () => {
      resolveResponse?.(
        new Response(
          JSON.stringify({
            data: [],
            page: { cursor: 'frontier', hasMore: false, limit: 40 },
          }),
          { status: 200 },
        ),
      );
    });
    await vi.waitFor(() =>
      expect(result?.progressiveSearch).toMatchObject({
        phase: 'complete',
        hasCompletedScanResponse: true,
      }),
    );
    expect(result?.runs).toEqual([]);
  });

  it('does not expose the previous result while a changed query awaits its first response', async () => {
    let resolveChangedQuery: ((response: Response) => void) | undefined;
    mocks.apiFetch.mockImplementation((pathname: string) => {
      if (pathname.includes('second-query')) {
        return new Promise<Response>((resolve) => {
          resolveChangedQuery = resolve;
        });
      }
      return Promise.resolve(
        new Response(
          JSON.stringify({
            data: [run('old-run')],
            page: { cursor: 'old-frontier', hasMore: false, limit: 40 },
          }),
          { status: 200 },
        ),
      );
    });

    let result: RunsPaginationResult | undefined;
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);

    await act(async () => {
      root?.render(
        <QueryClientProvider client={queryClient!}>
          <RunsPaginationHarness
            celQuery="event.data.userId == 'first-query'"
            onRender={(value) => (result = value)}
          />
        </QueryClientProvider>,
      );
    });
    await vi.waitFor(() =>
      expect(result?.progressiveSearch).toMatchObject({
        phase: 'complete',
        hasCompletedScanResponse: true,
      }),
    );
    expect(result?.runs.map(({ id }) => id)).toEqual(['old-run']);

    const changedQueryRenders: RunsPaginationResult[] = [];
    await act(async () => {
      root?.render(
        <QueryClientProvider client={queryClient!}>
          <RunsPaginationHarness
            celQuery="event.data.userId == 'second-query'"
            onRender={(value) => {
              result = value;
              changedQueryRenders.push(value);
            }}
          />
        </QueryClientProvider>,
      );
    });

    expect(resolveChangedQuery).toBeTypeOf('function');
    expect(changedQueryRenders.length).toBeGreaterThan(0);
    expect(
      changedQueryRenders.every(
        (rendered) =>
          rendered.runs.length === 0 &&
          rendered.progressiveSearch?.phase === 'searching' &&
          rendered.progressiveSearch.hasCompletedScanResponse === false,
      ),
    ).toBe(true);
  });

  it('does not start another page request after pagination fails', async () => {
    mocks.apiFetch.mockImplementation((pathname: string) => {
      if (pathname.includes('cursor=second-page')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({ errors: [{ message: 'Too many requests' }] }),
            { status: 429, headers: { 'Retry-After': '0' } },
          ),
        );
      }
      return Promise.resolve(
        new Response(
          JSON.stringify({
            data: [run('run-1')],
            page: { cursor: 'second-page', hasMore: true, limit: 40 },
          }),
          { status: 200 },
        ),
      );
    });

    let result: RunsPaginationResult | undefined;
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);

    await act(async () => {
      root?.render(
        <QueryClientProvider client={queryClient!}>
          <RunsPaginationHarness onRender={(value) => (result = value)} />
        </QueryClientProvider>,
      );
    });
    await vi.waitFor(() => expect(mocks.apiFetch).toHaveBeenCalledTimes(1));
    await vi.waitFor(() => expect(result?.hasNextPage).toBe(true));

    await act(async () => result?.loadMore());
    await vi.waitFor(() => expect(result?.error).toBeInstanceOf(Error));
    await vi.waitFor(() => expect(mocks.apiFetch).toHaveBeenCalledTimes(5));

    await act(async () => result?.loadMore());
    expect(mocks.apiFetch).toHaveBeenCalledTimes(5);
  });
});
