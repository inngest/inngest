// @vitest-environment jsdom

import { act } from 'react-dom/test-utils';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { Runs } from './Runs';

const mocks = vi.hoisted(() => ({
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
    booleanFlag: () => ({ isReady: true, value: true }),
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

describe('Runs REST readiness', () => {
  let root: Root | undefined;
  let container: HTMLDivElement | undefined;

  afterEach(async () => {
    if (root) await act(async () => root?.unmount());
    container?.remove();
    root = undefined;
    container = undefined;
    mocks.functionResult.data = undefined;
    mocks.functionResult.fetching = true;
    vi.clearAllMocks();
  });

  it('waits for function metadata before requesting REST runs', async () => {
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);

    await act(async () => {
      root?.render(<Runs scope="fn" functionSlug="app-function" />);
    });
    expect(mocks.useRunsPagination).toHaveBeenLastCalledWith(
      expect.objectContaining({ enabled: false }),
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
      expect.objectContaining({ enabled: true }),
    );
  });
});
