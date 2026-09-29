// @vitest-environment jsdom

import { act, cleanup, render } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { AppFilterDocument } from './queries';
import { Runs } from './Runs';

const mocks = vi.hoisted(() => ({
  appIDs: undefined as string[] | undefined,
  appsRefetch: vi.fn(),
  functionRefetch: vi.fn(),
  functionResult: {
    data: undefined as
      | {
          workspace: {
            workflow: {
              app: { externalID: string };
              isPaused: boolean;
              slug: string;
            };
          };
        }
      | undefined,
    error: undefined as Error | undefined,
    fetching: true,
  },
  runsPage: vi.fn((_props?: any) => null),
  searchParams: {} as Record<string, string | undefined>,
  useQuery: vi.fn((_opts?: unknown): any => [
    {
      data: undefined,
      error: undefined as Error | undefined,
      fetching: false,
    },
    vi.fn(),
  ]),
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

vi.mock('@inngest/components/hooks/useCalculatedStartTime', () => ({
  useCalculatedStartTime: ({ startTime }: { startTime?: string }) =>
    new Date(startTime ?? '2026-09-15T00:00:00Z'),
}));

vi.mock('@inngest/components/hooks/useSearchParams', () => ({
  useBooleanSearchParam: () => [undefined],
  useSearchParam: (key: string) => [mocks.searchParams[key]],
  useStringArraySearchParam: (key: string) => [
    key === 'filterApp' ? mocks.appIDs : undefined,
  ],
}));

vi.mock('@/components/Environments/environment-context', () => ({
  useEnvironment: () => ({ id: 'environment-id', slug: 'production' }),
}));

vi.mock('@/components/RunDetails/useGetTrigger', () => ({
  useGetTrigger: () => vi.fn(),
}));

vi.mock('@/queries/functions', () => ({
  useFunction: () => [mocks.functionResult, mocks.functionRefetch],
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

describe('Runs metadata', () => {
  afterEach(() => {
    cleanup();
    mocks.functionResult.data = undefined;
    mocks.functionResult.error = undefined;
    mocks.functionResult.fetching = true;
    mocks.appIDs = undefined;
    mocks.searchParams = {};
    vi.clearAllMocks();
  });

  it('waits for function metadata before requesting REST runs', () => {
    const { rerender } = render(
      <Runs scope="fn" functionSlug="app-function" />,
    );
    expect(mocks.useRunsPagination).toHaveBeenLastCalledWith(
      expect.objectContaining({ pause: true }),
    );
    expect(mocks.useQuery).toHaveBeenLastCalledWith(
      expect.objectContaining({ pause: true }),
    );

    mocks.functionResult.fetching = false;
    mocks.functionResult.data = {
      workspace: {
        workflow: {
          app: { externalID: 'app' },
          isPaused: false,
          slug: 'app-app-function',
        },
      },
    };
    rerender(<Runs scope="fn" functionSlug="app-function" />);
    expect(mocks.useRunsPagination).toHaveBeenLastCalledWith(
      expect.objectContaining({ pause: false }),
    );
    expect(mocks.useQuery).toHaveBeenLastCalledWith(
      expect.objectContaining({ pause: false }),
    );
  });

  it('surfaces function metadata failures instead of falling back', () => {
    const metadataError = new Error('metadata request failed');
    mocks.functionResult.fetching = false;
    mocks.functionResult.error = metadataError;
    render(<Runs scope="fn" functionSlug="app-function" />);

    expect(mocks.useRunsPagination).toHaveBeenLastCalledWith(
      expect.objectContaining({ pause: true }),
    );
    expect(mocks.runsPage.mock.lastCall?.[0]).toEqual(
      expect.objectContaining({
        error: metadataError,
        isLoadingInitial: false,
        isLoadingMore: false,
      }),
    );

    act(() => mocks.runsPage.mock.lastCall?.[0].onRefresh());
    expect(mocks.functionRefetch).toHaveBeenCalledWith({
      requestPolicy: 'network-only',
    });
  });

  it('retries app metadata failures', () => {
    const metadataError = new Error('app metadata request failed');
    mocks.appIDs = ['app-id'];
    mocks.useQuery.mockImplementationOnce(() => [
      { data: undefined, error: metadataError, fetching: false },
      mocks.appsRefetch,
    ]);

    render(<Runs scope="env" />);

    expect(mocks.runsPage.mock.lastCall?.[0]).toEqual(
      expect.objectContaining({ error: metadataError }),
    );
    act(() => mocks.runsPage.mock.lastCall?.[0].onRefresh());
    expect(mocks.appsRefetch).toHaveBeenCalledWith({
      requestPolicy: 'network-only',
    });
  });

  it('wires a progressive CEL search to Insights while omitting app filters', () => {
    mocks.appIDs = ['internal-app-id'];
    mocks.searchParams = {
      search: "event.data.customer == 'A&B'",
      start: '2026-02-03T04:05:06.789Z',
      end: '2026-02-04T07:08:09.123Z',
      timeField: 'STARTED_AT',
    };
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

    const { rerender } = render(<Runs scope="env" />);

    const props = mocks.runsPage.mock.lastCall?.[0];
    const href = props.progressiveSearch.insightsHref;
    const url = new URL(href, 'https://app.inngest.com');
    const sql = url.searchParams.get('sql');
    expect(url.pathname).toBe('/env/production/insights');
    expect(url.searchParams.get('name')).toBe('Runs search');
    expect(sql).toContain("cel(`event.data.customer == 'A&B'`)");
    expect(sql).toContain('queued_at > now() - INTERVAL 3 DAY');
    expect(sql).not.toContain('app_id');
    expect(sql).not.toContain('ORDER BY');
    expect(sql).not.toContain('LIMIT');

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
    rerender(<Runs scope="env" />);
    expect(mocks.runsPage.mock.lastCall?.[0].progressiveSearch).toBeUndefined();
  });

  it("waits for the function query's fully qualified slug for Insights", () => {
    mocks.searchParams = { search: 'event.data.ready == true' };
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

    const { rerender } = render(
      <Runs scope="fn" functionSlug="billing-app-charge-invoice" />,
    );
    expect(
      mocks.runsPage.mock.lastCall?.[0].progressiveSearch.insightsHref,
    ).toBeUndefined();

    mocks.functionResult.fetching = false;
    mocks.functionResult.data = {
      workspace: {
        workflow: {
          app: { externalID: 'billing-app' },
          isPaused: false,
          slug: 'billing-app-charge-invoice',
        },
      },
    };
    rerender(<Runs scope="fn" functionSlug="billing-app-charge-invoice" />);

    const href =
      mocks.runsPage.mock.lastCall?.[0].progressiveSearch.insightsHref;
    const sql = new URL(href, 'https://app.inngest.com').searchParams.get(
      'sql',
    );
    expect(sql).toContain("function_id = 'billing-app-charge-invoice'");
  });
});
