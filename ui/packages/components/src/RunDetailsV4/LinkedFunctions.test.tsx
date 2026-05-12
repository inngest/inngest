import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { RunDeferSummary, RunDeferredFromSummary } from '../SharedContext/useGetRun';
import { LinkedFunctions } from './LinkedFunctions';
import type { InvokedRun } from './runDetailsUtils';

// @inngest/components/* self-imports don't resolve in vitest without a workspace alias.
vi.mock('../Link', () => ({
  Link: ({ href, children }: { href?: string; children: React.ReactNode }) => (
    <a href={href}>{children}</a>
  ),
}));

vi.mock('../SharedContext/usePathCreator', () => ({
  usePathCreator: () => ({
    pathCreator: {
      runPopout: ({ runID }: { runID: string }) => `/runs/${runID}`,
      function: ({ functionSlug }: { functionSlug: string }) => `/functions/${functionSlug}`,
    },
  }),
}));

vi.mock('../Table/Cell', () => ({
  IDCell: ({ children }: { children: React.ReactNode }) => (
    <span data-testid="id-cell">{children}</span>
  ),
  StatusCell: ({ status }: { status: string }) => <span data-testid="status-cell">{status}</span>,
}));

vi.mock('../Tooltip/OptionalTooltip', () => ({
  OptionalTooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

afterEach(() => {
  cleanup();
});

function makeDefer(overrides: Partial<RunDeferSummary> = {}): RunDeferSummary {
  return {
    id: 'hash-1',
    userDeferID: 'user-id-1',
    fnSlug: 'child-fn',
    status: 'SCHEDULED',
    run: null,
    ...overrides,
  };
}

function makeInvoked(overrides: Partial<InvokedRun> = {}): InvokedRun {
  return {
    spanID: 'span-1',
    invokerName: 'parent.step',
    functionID: 'invoked-fn',
    runID: '01INVOKED01',
    status: 'COMPLETED',
    ...overrides,
  };
}

describe('LinkedFunctions', () => {
  it('renders no section headers when all inputs are empty', () => {
    render(<LinkedFunctions runID="run-self" invoked={[]} />);
    expect(screen.queryByText('Parent function')).toBeNull();
    expect(screen.queryByText('Deferred functions')).toBeNull();
    expect(screen.queryByText('Parallel defers')).toBeNull();
    expect(screen.queryByText('Invoked functions')).toBeNull();
  });

  it('renders only the Parent section when only deferredFrom is set', () => {
    const deferredFrom: RunDeferredFromSummary = {
      parentRunID: '01PARENT01',
      parentFnSlug: 'parent-fn',
      parentRun: null,
    };
    render(<LinkedFunctions runID="run-self" invoked={[]} deferredFrom={deferredFrom} />);
    expect(screen.getByText('Parent function')).toBeTruthy();
    expect(screen.queryByText('Deferred functions')).toBeNull();
    expect(screen.queryByText('Parallel defers')).toBeNull();
    expect(screen.queryByText('Invoked functions')).toBeNull();
  });

  it('renders only the Deferred section when only defers is set', () => {
    render(<LinkedFunctions runID="run-self" invoked={[]} defers={[makeDefer()]} />);
    expect(screen.getByText('Deferred functions')).toBeTruthy();
    expect(screen.queryByText('Parent function')).toBeNull();
    expect(screen.queryByText('Parallel defers')).toBeNull();
    expect(screen.queryByText('Invoked functions')).toBeNull();
  });

  it('renders only the Invoked section when only invoked is set', () => {
    render(<LinkedFunctions runID="run-self" invoked={[makeInvoked()]} />);
    expect(screen.getByText('Invoked functions')).toBeTruthy();
    expect(screen.queryByText('Parent function')).toBeNull();
    expect(screen.queryByText('Deferred functions')).toBeNull();
    expect(screen.queryByText('Parallel defers')).toBeNull();
  });

  it('parallel defers exclude the current run', () => {
    const sibling = makeDefer({
      id: 'hash-sibling',
      userDeferID: 'user-sibling',
      run: {
        id: 'run-sibling',
        status: 'COMPLETED',
        function: { name: 'Sibling Fn', slug: 'sibling-fn' },
      },
    });
    const self = makeDefer({
      id: 'hash-self',
      userDeferID: 'user-self',
      run: {
        id: 'run-self',
        status: 'COMPLETED',
        function: { name: 'Self Fn', slug: 'self-fn' },
      },
    });
    const deferredFrom: RunDeferredFromSummary = {
      parentRunID: '01PARENT01',
      parentFnSlug: 'parent-fn',
      parentRun: {
        id: '01PARENT01',
        status: 'COMPLETED',
        function: { name: 'Parent Fn', slug: 'parent-fn' },
        defers: [self, sibling],
      },
    };
    render(<LinkedFunctions runID="run-self" invoked={[]} deferredFrom={deferredFrom} />);
    expect(screen.getByText('Parallel defers')).toBeTruthy();
    // Sibling is shown.
    expect(screen.getByText('user-sibling')).toBeTruthy();
    // The current run's userDeferID does not appear in the parallel list.
    expect(screen.queryByText('user-self')).toBeNull();
  });

  it('hides the Parallel defers section when the filter empties the list', () => {
    const self = makeDefer({
      id: 'hash-self',
      userDeferID: 'user-self',
      run: {
        id: 'run-self',
        status: 'COMPLETED',
        function: { name: 'Self Fn', slug: 'self-fn' },
      },
    });
    const deferredFrom: RunDeferredFromSummary = {
      parentRunID: '01PARENT01',
      parentFnSlug: 'parent-fn',
      parentRun: {
        id: '01PARENT01',
        status: 'COMPLETED',
        function: { name: 'Parent Fn', slug: 'parent-fn' },
        defers: [self],
      },
    };
    render(<LinkedFunctions runID="run-self" invoked={[]} deferredFrom={deferredFrom} />);
    expect(screen.queryByText('Parallel defers')).toBeNull();
  });

  it('renders the userDeferID, not the hashed id', () => {
    render(
      <LinkedFunctions
        runID="run-self"
        invoked={[]}
        defers={[makeDefer({ id: 'sha1-hashed-id', userDeferID: 'order-7' })]}
      />
    );
    expect(screen.getByText('order-7')).toBeTruthy();
    expect(screen.queryByText('sha1-hashed-id')).toBeNull();
  });

  it('falls back to fnSlug for the function name when run is null', () => {
    render(
      <LinkedFunctions
        runID="run-self"
        invoked={[]}
        defers={[makeDefer({ run: null, fnSlug: 'fallback-fn' })]}
      />
    );
    expect(screen.getByText('fallback-fn')).toBeTruthy();
  });

  it("shows '-' in the run-ID column when the run is null", () => {
    render(<LinkedFunctions runID="run-self" invoked={[]} defers={[makeDefer({ run: null })]} />);
    // We only render a '-' for the missing run cell. Status and other cells are
    // present too but `-` should appear at least once.
    const dashes = screen.getAllByText('-');
    expect(dashes.length).toBeGreaterThan(0);
  });

  it('prefers the run status over the defer-row status when a run is linked', () => {
    render(
      <LinkedFunctions
        runID="run-self"
        invoked={[]}
        defers={[
          makeDefer({
            status: 'SCHEDULED',
            run: {
              id: '01CHILDRUN01',
              status: 'COMPLETED',
              function: { name: 'Child Fn', slug: 'child-fn' },
            },
          }),
        ]}
      />
    );
    const statusCells = screen.getAllByTestId('status-cell');
    const statuses = statusCells.map((el) => el.textContent);
    expect(statuses).toContain('COMPLETED');
    expect(statuses).not.toContain('SCHEDULED');
  });
});
