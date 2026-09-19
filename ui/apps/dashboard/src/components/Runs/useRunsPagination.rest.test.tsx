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
  gqlRefetch: vi.fn(),
}));

vi.mock('@/queries/useInngestAPIFetch', () => ({
  useInngestAPIFetch: () => mocks.apiFetch,
}));

vi.mock('urql', () => ({
  useQuery: () => [
    { data: undefined, error: undefined, fetching: false },
    mocks.gqlRefetch,
  ],
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
}: {
  onRender: (result: RunsPaginationResult) => void;
}) {
  onRender(
    useRunsPagination({
      commonQueryVars,
      tracePreviewEnabled: false,
      shouldUseREST: true,
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
});
