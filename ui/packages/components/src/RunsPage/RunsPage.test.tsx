import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TooltipProvider } from '../Tooltip/Tooltip';
import { RunsPage } from './RunsPage';

vi.mock('../CodeSearch/CodeSearch', () => ({ default: () => null }));
vi.mock('../Filter/EntityFilter', () => ({ default: () => null }));
vi.mock('../RunDetailsV4', () => ({ RunDetailsV4: () => null }));
vi.mock('../hooks/useSearchParams', () => ({
  useBatchedSearchParams: () => vi.fn(),
  useBooleanSearchParam: () => [false, vi.fn(), vi.fn()],
  useSearchParam: () => [undefined, vi.fn(), vi.fn()],
  useStringArraySearchParam: () => [[], vi.fn(), vi.fn()],
  useValidatedArraySearchParam: () => [[], vi.fn(), vi.fn()],
  useValidatedSearchParam: () => [undefined, vi.fn()],
}));
vi.mock('./RunsStatusFilter', () => ({ default: () => null }));
vi.mock('./RunsTable', () => ({ default: () => null }));
vi.mock('./RunsTypeFilter', () => ({ default: () => null }));
vi.mock('./columns', () => ({
  isColumnID: () => false,
  useScopedColumns: () => [],
}));
vi.mock('@inngest/components/Filter/TimeFieldFilter', () => ({
  default: () => null,
}));
vi.mock('@inngest/components/Filter/TimeFilter', () => ({
  TimeFilter: () => null,
}));
vi.mock('@inngest/components/Table', () => ({ TableFilter: () => null }));

afterEach(cleanup);

describe('RunsPage progressive search actions', () => {
  const baseProps = {
    data: [],
    features: { history: 7, tracesPreview: true, isDeferred: true },
    getTrigger: vi.fn(),
    hasMore: false,
    isLoadingInitial: false,
    isLoadingMore: false,
    onScrollToTop: vi.fn(),
    scope: 'env' as const,
    totalCount: undefined,
  };

  it('shows the Insights action only while progressive search state exists', () => {
    const { rerender } = render(
      <TooltipProvider>
        <RunsPage
          {...baseProps}
          progressiveSearch={{
            phase: 'searching',
            hasCompletedScanResponse: true,
            cancel: vi.fn(),
            resume: vi.fn(),
            insightsHref: '/env/production/insights?sql=SELECT+%2A+FROM+runs',
          }}
        />
      </TooltipProvider>
    );

    const action = screen.getByText('Use Insights').closest('a');
    expect(action?.getAttribute('href')).toBe('/env/production/insights?sql=SELECT+%2A+FROM+runs');

    rerender(
      <TooltipProvider>
        <RunsPage
          {...baseProps}
          progressiveSearch={{
            phase: 'searching',
            hasCompletedScanResponse: true,
            cancel: vi.fn(),
            resume: vi.fn(),
          }}
        />
      </TooltipProvider>
    );
    expect(screen.queryByText('Use Insights')).toBeNull();

    rerender(
      <TooltipProvider>
        <RunsPage {...baseProps} />
      </TooltipProvider>
    );
    expect(screen.queryByText('Use Insights')).toBeNull();
  });
});
