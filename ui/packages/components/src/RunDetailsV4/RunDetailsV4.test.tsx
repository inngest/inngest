/**
 * RunDetailsV4 regression tests for issue #3844.
 *
 * Verifies that the detail view uses trace.status (not runData.status)
 * when syncing status back to the run list via updateDynamicRunData.
 */

import { cleanup, render } from '@testing-library/react';
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';

// --- Mocks ---

const mockUpdateDynamicRunData = vi.fn();

vi.mock('../SharedContext/useBooleanFlag', () => ({
  useBooleanFlag: () => ({
    booleanFlag: () => ({ isReady: true, value: false }),
  }),
}));

vi.mock('../SharedContext/useGetTraceResult', () => ({
  useGetTraceResult: () => ({ data: undefined, error: undefined, refetch: vi.fn() }),
}));

vi.mock('./runDetailsUtils', async (importOriginal) => {
  const actual = (await importOriginal()) as Record<string, unknown>;
  return {
    ...actual,
    useDynamicRunData: () => ({
      dynamicRunData: undefined,
      updateDynamicRunData: mockUpdateDynamicRunData,
    }),
    useStepSelection: () => ({ selectedStep: undefined, selectStep: vi.fn() }),
  };
});

vi.mock('../Table/Cell', () => ({
  StatusCell: ({ status }: { status: string }) => <span data-testid="status-cell">{status}</span>,
}));

vi.mock('../TriggerDetails', () => ({
  TriggerDetails: () => null,
}));

vi.mock('../Error/ErrorCard', () => ({
  ErrorCard: () => null,
}));

vi.mock('./RunInfo', () => ({
  RunInfo: () => <div data-testid="run-info" />,
}));

vi.mock('./StepInfo', () => ({
  StepInfo: () => null,
}));

vi.mock('./Tabs', () => ({
  Tabs: () => <div data-testid="tabs" />,
}));

vi.mock('./Timeline', () => ({
  Timeline: () => null,
}));

vi.mock('./TopInfo', () => ({
  TopInfo: () => <div data-testid="top-info" />,
}));

vi.mock('./Waiting', () => ({
  Waiting: () => null,
}));

vi.mock('../icons/DragDivider', () => ({
  DragDivider: () => null,
}));

// jsdom doesn't provide ResizeObserver
beforeAll(() => {
  global.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('RunDetailsV4 - issue #3844 status sync', () => {
  it('calls updateDynamicRunData with trace.status, not runData.status', async () => {
    // Mock useGetRun to return divergent statuses
    const { useGetRun: _orig, ...rest } = await import('../SharedContext/useGetRun');
    vi.doMock('../SharedContext/useGetRun', () => ({
      ...rest,
      useGetRun: () => ({
        data: {
          id: 'run-1',
          status: 'RUNNING',
          fn: { id: 'fn-1', name: 'Test Fn', slug: 'test-fn' },
          app: { externalID: 'app-1', name: 'Test App' },
          hasAI: false,
          trace: {
            status: 'COMPLETED',
            spanID: 'span-1',
            traceID: 'trace-1',
            isRoot: true,
            name: 'root',
            endedAt: '2024-01-01T00:00:10Z',
            childrenSpans: [],
          },
        },
        error: undefined,
        loading: false,
        refetch: vi.fn(),
      }),
    }));

    // Re-import after mocking
    const { RunDetailsV4 } = await import('./RunDetailsV4');

    render(
      <RunDetailsV4
        standalone={false}
        runID="run-1"
        getTrigger={vi.fn() as any}
        initialRunData={{ status: 'RUNNING' } as any}
      />
    );

    // updateDynamicRunData should have been called with trace.status = 'COMPLETED'
    expect(mockUpdateDynamicRunData).toHaveBeenCalledWith(
      expect.objectContaining({
        runID: 'run-1',
        status: 'COMPLETED',
      })
    );

    // Crucially, it must NOT have been called with 'RUNNING'
    const calls = mockUpdateDynamicRunData.mock.calls;
    for (const call of calls) {
      expect(call[0].status).not.toBe('RUNNING');
    }
  });
});
