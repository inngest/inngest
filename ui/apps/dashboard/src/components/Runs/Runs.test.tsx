// @vitest-environment jsdom

import { act } from 'react-dom/test-utils';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { AppFilterDocument } from './queries';
import { Runs } from './Runs';

const mocks = vi.hoisted(() => ({
  flag: { isReady: false, value: false },
  functionResult: {
    data: undefined as
      | {
          workspace: {
            workflow: {
              app: { externalID: string };
              isPaused: boolean;
            };
          };
        }
      | undefined,
    fetching: true,
  },
  useQuery: vi.fn((_opts?: unknown): any => [
    { data: undefined, error: undefined, fetching: false },
    vi.fn(),
  ]),
  runsPage: vi.fn((_props?: any) => null),
  searchParams: {} as Record<string, string | undefined>,
  stringArraySearchParams: {} as Record<string, string[] | undefined>,
  useRunsPagination: vi.fn((_opts?: unknown): any => ({
    runs: [],
    isLoadingInitial: false,
    isLoadingMore: false,
    hasNextPage: false,
    loadMore: vi.fn(),
    reset: vi.fn(),
    error: undefined,
    progressiveSearch: undefined,
  })),
}));

vi.mock('urql', async (importOriginal) => ({
  ...(await importOriginal<typeof import('urql')>()),
  useQuery: mocks.useQuery,
}));

vi.mock(
  '@inngest/components/InfiniteScrollTrigger/InfiniteScrollTrigger',
  () => ({
    InfiniteScrollTrigger: () => null,
  }),
);

vi.mock('@inngest/components/RunsPage/RunsPage', () => ({
  RunsPage: mocks.runsPage,
}));

vi.mock('@inngest/components/SharedContext/useBooleanFlag', () => ({
  useBooleanFlag: () => ({
    booleanFlag: (flag: string) =>
      flag === 'rest-runs-table' ? mocks.flag : { isReady: true, value: true },
  }),
}));

vi.mock('@inngest/components/hooks/useCalculatedStartTime', () => ({
  useCalculatedStartTime: ({ startTime }: { startTime?: string }) =>
    new Date(startTime ?? '2026-09-15T00:00:00Z'),
}));

vi.mock('@inngest/components/hooks/useSearchParams', () => ({
  useBooleanSearchParam: () => [undefined],
  useSearchParam: (name: string) => [mocks.searchParams[name]],
  useStringArraySearchParam: (name: string) => [
    mocks.stringArraySearchParams[name],
  ],
}));

vi.mock('@/components/Environments/environment-context', () => ({
  useEnvironment: () => ({ id: 'environment-id', slug: 'production' }),
}));

vi.mock('@/components/RunDetails/useGetTrigger', () => ({
  useGetTrigger: () => vi.fn(),
}));

vi.mock('@/queries/functions', () => ({
  useFunction: () => [mocks.functionResult],
}));

vi.mock('@/utils/useAccountFeatures', () => ({
  useAccountFeatures: () => ({ data: { history: 7 } }),
}));

vi.mock('./AccountConcurrencyBanner', () => ({
  AccountConcurrencyBanner: () => null,
}));

vi.mock('./useRunsPagination', () => ({
  useRunsPagination: mocks.useRunsPagination,
}));

describe('Runs transport selection', () => {
  let root: Root | undefined;
  let container: HTMLDivElement | undefined;

  afterEach(async () => {
    if (root) await act(async () => root?.unmount());
    container?.remove();
    root = undefined;
    container = undefined;
    mocks.flag.isReady = false;
    mocks.flag.value = false;
    mocks.functionResult.data = undefined;
    mocks.functionResult.fetching = true;
    mocks.searchParams = {};
    mocks.stringArraySearchParams = {};
    vi.clearAllMocks();
  });

  it('waits for the REST flag and function metadata before choosing a transport', async () => {
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);

    await act(async () => {
      root?.render(<Runs scope="fn" functionSlug="app-function" />);
    });
    expect(mocks.useRunsPagination).toHaveBeenLastCalledWith(
      expect.objectContaining({ pause: true, shouldUseREST: false }),
    );
    expect(mocks.useQuery).toHaveBeenLastCalledWith(
      expect.objectContaining({ pause: true }),
    );

    mocks.flag.isReady = true;
    mocks.flag.value = true;
    await act(async () => {
      root?.render(<Runs scope="fn" functionSlug="app-function" />);
    });
    expect(mocks.useRunsPagination).toHaveBeenLastCalledWith(
      expect.objectContaining({ pause: true, shouldUseREST: false }),
    );
    expect(mocks.useQuery).toHaveBeenLastCalledWith(
      expect.objectContaining({ pause: true }),
    );

    mocks.functionResult.fetching = false;
    mocks.functionResult.data = {
      workspace: {
        workflow: { app: { externalID: 'app' }, isPaused: false },
      },
    };
    await act(async () => {
      root?.render(<Runs scope="fn" functionSlug="app-function" />);
    });
    expect(mocks.useRunsPagination).toHaveBeenLastCalledWith(
      expect.objectContaining({ pause: false, shouldUseREST: true }),
    );
    expect(mocks.useQuery).toHaveBeenLastCalledWith(
      expect.objectContaining({ pause: false }),
    );

    mocks.flag.value = false;
    await act(async () => {
      root?.render(<Runs scope="fn" functionSlug="app-function" />);
    });
    expect(mocks.useRunsPagination).toHaveBeenLastCalledWith(
      expect.objectContaining({ pause: false, shouldUseREST: false }),
    );
  });

  it('only exposes an Insights handoff for a progressive CEL search', async () => {
    mocks.flag.isReady = true;
    mocks.flag.value = true;
    mocks.searchParams = {
      search: "event.data.customer == 'A&B'",
      start: '2026-02-03T04:05:06.789Z',
      end: '2026-02-04T07:08:09.123Z',
      timeField: 'STARTED_AT',
    };
    mocks.stringArraySearchParams = { filterApp: ['internal-app-id'] };
    mocks.useQuery.mockImplementation((options: unknown) => [
      (options as { query?: unknown }).query === AppFilterDocument
        ? {
            data: {
              env: {
                apps: [
                  {
                    id: 'internal-app-id',
                    externalID: 'payments-api',
                  },
                ],
              },
            },
            error: undefined,
            fetching: false,
          }
        : { data: undefined, error: undefined, fetching: false },
      vi.fn(),
    ]);
    mocks.useRunsPagination.mockReturnValue({
      runs: [],
      isLoadingInitial: false,
      isLoadingMore: false,
      hasNextPage: false,
      loadMore: vi.fn(),
      reset: vi.fn(),
      error: undefined,
      progressiveSearch: {
        phase: 'searching',
        hasCompletedScanResponse: true,
        cursor: undefined,
        cancel: vi.fn(),
        resume: vi.fn(),
      },
    });

    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
    await act(async () => {
      root?.render(<Runs scope="env" />);
    });

    const props = mocks.runsPage.mock.lastCall?.[0];
    expect(props.progressiveSearch.insightsHref).toContain(
      '/env/production/insights?',
    );
    const sql = new URL(
      props.progressiveSearch.insightsHref,
      'https://app.inngest.com',
    ).searchParams.get('sql');
    expect(sql).toContain("cel(`event.data.customer == 'A&B'`)");
    expect(sql).toContain("started_at >= '2026-02-03 04:05:06.789'");
    expect(sql).toContain("app_id IN ('payments-api')");

    mocks.useRunsPagination.mockReturnValue({
      runs: [],
      isLoadingInitial: false,
      isLoadingMore: false,
      hasNextPage: false,
      loadMore: vi.fn(),
      reset: vi.fn(),
      error: undefined,
      progressiveSearch: undefined,
    });
    await act(async () => {
      root?.render(<Runs scope="env" />);
    });
    expect(mocks.runsPage.mock.lastCall?.[0].progressiveSearch).toBeUndefined();
  });
});
