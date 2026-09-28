// @vitest-environment jsdom

import { act } from 'react-dom/test-utils';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';

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
  useQuery: vi.fn(() => [
    { data: undefined, error: undefined, fetching: false },
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
  RunsPage: () => null,
}));

vi.mock('@inngest/components/SharedContext/useBooleanFlag', () => ({
  useBooleanFlag: () => ({
    booleanFlag: (flag: string) =>
      flag === 'rest-runs-table' ? mocks.flag : { isReady: true, value: true },
  }),
}));

vi.mock('@inngest/components/hooks/useCalculatedStartTime', () => ({
  useCalculatedStartTime: () => new Date('2026-09-15T00:00:00Z'),
}));

vi.mock('@inngest/components/hooks/useSearchParams', () => ({
  useBooleanSearchParam: () => [undefined],
  useSearchParam: () => [undefined],
  useStringArraySearchParam: () => [undefined],
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
});
