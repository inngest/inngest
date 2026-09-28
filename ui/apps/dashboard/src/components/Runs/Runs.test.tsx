// @vitest-environment jsdom

import { act, cleanup, render } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

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
            };
          };
        }
      | undefined,
    error: undefined as Error | undefined,
    fetching: true,
  },
  runsPage: vi.fn((_props: { error?: Error; onRefresh: () => void }) => null),
  useQuery: vi.fn(() => [
    {
      data: undefined,
      error: undefined as Error | undefined,
      fetching: false,
    },
    vi.fn(),
  ]),
  useRunsPagination: vi.fn(() => ({
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
  useCalculatedStartTime: () => new Date('2026-09-15T00:00:00Z'),
}));

vi.mock('@inngest/components/hooks/useSearchParams', () => ({
  useBooleanSearchParam: () => [undefined],
  useSearchParam: () => [undefined],
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
        workflow: { app: { externalID: 'app' }, isPaused: false },
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
});
