// @vitest-environment jsdom
import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ExperimentDetailPage } from './ExperimentDetailPage';

const mocks = vi.hoisted(() => ({
  insightsSQL: 'x'.repeat(8 * 1024 + 1),
  toastError: vi.fn(),
  trackOpenedInInsights: vi.fn(),
}));

vi.mock('sonner', () => ({ toast: { error: mocks.toastError } }));
vi.mock('urql', () => ({
  useQuery: () => [
    { data: { account: { entitlements: { history: { limit: 30 } } } } },
  ],
}));
vi.mock('@/components/Environments/environment-context', () => ({
  useEnvironment: () => ({ slug: 'production' }),
}));
vi.mock('@inngest/components/Header/Header', () => ({ Header: () => null }));
vi.mock('@/components/Experiments/useExperiments', () => ({
  useExperimentDetail: () => ({
    data: {
      variants: [{ variantName: 'asymmetric-B', runCount: 17, metrics: [] }],
    },
    error: undefined,
    isPending: false,
    refetch: vi.fn(),
  }),
  useExperimentInsightsQuery: () => ({ data: mocks.insightsSQL }),
}));
vi.mock('@/components/Experiments/useScoringConfig', () => ({
  useScoringConfig: () => ({
    error: undefined,
    isPending: false,
    metrics: [],
    refetch: vi.fn(),
    updateMetric: vi.fn(),
  }),
}));
vi.mock('@/components/Experiments/VariantsTable', () => ({
  VariantsTable: ({ onOpenInsights }: { onOpenInsights: () => void }) => (
    <button onClick={onOpenInsights}>Open in Insights</button>
  ),
}));
vi.mock('@/components/Experiments/ExperimentDetailToolbar', () => ({
  ExperimentDetailToolbar: () => null,
}));
vi.mock('@/components/Experiments/RunCountDonutCard', () => ({
  RunCountDonutCard: () => null,
}));
vi.mock('@/components/Experiments/ScoreSummaryCard', () => ({
  ScoreSummaryCard: () => null,
}));
vi.mock('@/components/Experiments/InfoSidebar', () => ({
  InfoSidebar: () => null,
}));
vi.mock('@/components/Experiments/ScoringFormulaSidebar', () => ({
  ScoringFormulaSidebar: () => null,
}));
vi.mock('@/utils/analyticsEvents', () => ({
  trackOpenedInInsights: mocks.trackOpenedInInsights,
}));

describe('ExperimentDetailPage Insights action', () => {
  beforeEach(() => {
    mocks.toastError.mockReset();
    mocks.trackOpenedInInsights.mockReset();
    vi.spyOn(window, 'open').mockImplementation(() => null);
  });

  it('reports oversized generated SQL without opening or tracking a handoff', () => {
    render(
      <ExperimentDetailPage
        experimentName="checkout-ranking"
        functionID="function-asymmetric"
        functionName="Checkout"
        functionSlug="checkout"
        timeRange={{ type: 'live', durationMs: 3_600_000, preset: null }}
        hasTimeRangeSearch
        onTimeRangeChange={vi.fn()}
        selectedVariants={[]}
        onSelectedVariantsChange={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Open in Insights' }));

    expect(mocks.toastError).toHaveBeenCalledWith(
      'This query is too large to open in Insights.',
    );
    expect(window.open).not.toHaveBeenCalled();
    expect(mocks.trackOpenedInInsights).not.toHaveBeenCalled();
  });
});
