// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react';
import { RunDetailsV4 } from '@inngest/components/RunDetailsV4';
import type { GetRunData } from '@inngest/components/SharedContext/useGetRun';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({ run: undefined as GetRunData | undefined }));
vi.mock('@inngest/components/SharedContext/useGetRun', () => ({
  useGetRun: () => ({ data: mocks.run }),
}));
vi.mock('@inngest/components/SharedContext/useGetRunLinkage', () => ({
  useGetRunLinkage: () => ({}),
}));
vi.mock('@inngest/components/SharedContext/useGetTraceResult', () => ({
  useGetTraceResult: () => ({}),
}));
vi.mock('@inngest/components/SharedContext/useBooleanFlag', () => ({
  useBooleanFlag: () => ({
    booleanFlag: () => ({ value: false, isReady: true }),
  }),
}));
vi.mock('@inngest/components/RunDetailsV4/runDetailsUtils', () => ({
  useDynamicRunData: () => ({ updateDynamicRunData: vi.fn() }),
  useStepSelection: () => ({ selectedStep: undefined }),
}));
vi.mock('@inngest/components/RunDetailsV4/RunInfo', () => ({
  RunInfo: ({ additionalAction }: { additionalAction?: React.ReactNode }) => (
    <div>{additionalAction}</div>
  ),
}));
vi.mock('@inngest/components/RunDetailsV4/Tabs', () => ({ Tabs: () => null }));
vi.mock('@inngest/components/RunDetailsV4/StepInfo', () => ({
  StepInfo: () => null,
}));
vi.mock('@inngest/components/RunDetailsV4/TopInfo', () => ({
  TopInfo: () => null,
}));
vi.mock('@inngest/components/Table/Cell', () => ({ StatusCell: () => null }));

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      disconnect() {}
    }
  );
  mocks.run = {
    id: 'run-1',
    status: 'FAILED',
    fn: { name: 'Example' },
    trace: { endedAt: '2026-10-21T00:00:00Z' },
  } as GetRunData;
});
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe('run details action slot', () => {
  it.each([true, false])(
    'supplies loaded run context (standalone=%s)',
    (standalone) => {
      const renderRunAction = vi.fn(() => <button>Debug with AI</button>);
      render(
        <RunDetailsV4
          standalone={standalone}
          runID="run-1"
          getTrigger={vi.fn()}
          renderRunAction={renderRunAction}
        />
      );
      expect(renderRunAction).toHaveBeenCalledWith(mocks.run, standalone);
      expect(
        screen.getByRole('button', { name: 'Debug with AI' })
      ).toBeTruthy();
    }
  );

  it.each(['public', 'loading', 'stale'] as const)(
    'does not render an action for %s context',
    (state) => {
      if (state === 'loading') mocks.run = undefined;
      if (state === 'stale') mocks.run = { ...mocks.run!, id: 'previous-run' };
      const renderRunAction = vi.fn(() => <button>Debug with AI</button>);
      render(
        <RunDetailsV4
          standalone={false}
          runID="run-1"
          getTrigger={vi.fn()}
          readOnly={state === 'public'}
          renderRunAction={renderRunAction}
        />
      );
      expect(renderRunAction).not.toHaveBeenCalled();
      expect(
        screen.queryByRole('button', { name: 'Debug with AI' })
      ).toBeNull();
    }
  );
});
