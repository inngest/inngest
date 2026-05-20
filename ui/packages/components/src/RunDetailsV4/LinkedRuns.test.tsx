import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type {
  RunDeferSummary,
  RunDeferredFromSummary,
  RunInvokedFromSummary,
} from '../SharedContext/useGetRunLinkage';
import { LinkedRuns } from './LinkedRuns';

// @inngest/components/* self-imports don't resolve in vitest without a workspace
// alias. Mock the routing primitives and pass through the Table cells / tooltip
// so assertions can target real rendered content (text, roles, hrefs) rather
// than mock-emitted test-ids.
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
  IDCell: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
  StatusCell: ({ status, label }: { status: string; label?: string }) => (
    <span>{label || status}</span>
  ),
  PillCell: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
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

describe('LinkedRuns', () => {
  it('renders Deferred + Invoked sections for a primary run (no deferredFrom)', () => {
    render(
      <LinkedRuns
        runID="run-self"
        invoked={[
          {
            spanID: 'span-1',
            invokerName: 'invoker-step',
            runID: 'run-invoked',
            functionID: 'invoked-fn',
            status: 'COMPLETED',
          },
        ]}
        defers={[makeDefer()]}
      />
    );
    expect(screen.getByText('Deferred runs')).toBeTruthy();
    expect(screen.getByText('Invoked runs')).toBeTruthy();
    expect(screen.queryByText('Parent run')).toBeNull();
    expect(screen.queryByText('Parallel defers')).toBeNull();
  });

  it('skips empty Deferred and Invoked sections for a primary run', () => {
    render(<LinkedRuns runID="run-self" invoked={[]} />);
    expect(screen.queryByText('Deferred runs')).toBeNull();
    expect(screen.queryByText('Invoked runs')).toBeNull();
  });

  it('renders Parent section and skips empty Parallel defers for a deferred run', () => {
    const deferredFrom: RunDeferredFromSummary = {
      parentRunID: '01PARENT01',
      parentRun: null,
    };
    render(<LinkedRuns runID="run-self" invoked={[]} deferredFrom={deferredFrom} />);
    expect(screen.getByText('Parent run')).toBeTruthy();
    expect(screen.queryByText('Parallel defers')).toBeNull();
    expect(screen.queryByText('Deferred runs')).toBeNull();
    expect(screen.queryByText('Invoked runs')).toBeNull();
  });

  it('renders no function pill when the parent run is null', () => {
    const deferredFrom: RunDeferredFromSummary = {
      parentRunID: '01PARENT01',
      parentRun: null,
    };
    render(<LinkedRuns runID="run-self" invoked={[]} deferredFrom={deferredFrom} />);
    const links = screen.getAllByRole('link');
    expect(links).toHaveLength(1);
    expect(links[0]?.getAttribute('href')).toBe('/runs/01PARENT01');
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
      parentRun: {
        id: '01PARENT01',
        status: 'COMPLETED',
        function: { name: 'Parent Fn', slug: 'parent-fn' },
        defers: [self, sibling],
      },
    };
    render(<LinkedRuns runID="run-self" invoked={[]} deferredFrom={deferredFrom} />);
    expect(screen.getByText('Parallel defers')).toBeTruthy();
    // Sibling is shown.
    expect(screen.getByText('user-sibling')).toBeTruthy();
    // The current run's userDeferID does not appear in the parallel list.
    expect(screen.queryByText('user-self')).toBeNull();
  });

  it('renders the userDeferID, not the hashed id', () => {
    render(
      <LinkedRuns
        runID="run-self"
        invoked={[]}
        defers={[makeDefer({ id: 'sha1-hashed-id', userDeferID: 'order-7' })]}
      />
    );
    expect(screen.getByText('order-7')).toBeTruthy();
    expect(screen.queryByText('sha1-hashed-id')).toBeNull();
  });

  it('falls back to fnSlug for the function pill when run is null', () => {
    render(
      <LinkedRuns
        runID="run-self"
        invoked={[]}
        defers={[makeDefer({ run: null, fnSlug: 'fallback-fn' })]}
      />
    );
    expect(screen.getByText('fallback-fn')).toBeTruthy();
  });

  it("shows '-' in the run-ID column when the run is null", () => {
    render(<LinkedRuns runID="run-self" invoked={[]} defers={[makeDefer({ run: null })]} />);
    // We only render a '-' for the missing run cell. Status and other cells are
    // present too but `-` should appear at least once.
    const dashes = screen.getAllByText('-');
    expect(dashes.length).toBeGreaterThan(0);
  });

  it('prefers the run status over the defer-row status when a run is linked', () => {
    render(
      <LinkedRuns
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
    expect(screen.getByText('COMPLETED')).toBeTruthy();
    expect(screen.queryByText('SCHEDULED')).toBeNull();
  });

  it('renders Invoked by section when invokedFrom is set', () => {
    const invokedFrom: RunInvokedFromSummary = {
      parentRunID: '01INVOKER01',
      parentRun: {
        id: '01INVOKER01',
        status: 'COMPLETED',
        function: { name: 'Invoker Fn', slug: 'invoker-fn' },
      },
      stepName: 'invoke-child',
    };
    render(<LinkedRuns runID="run-self" invoked={[]} invokedFrom={invokedFrom} />);
    expect(screen.getByText('Invoked by')).toBeTruthy();
    expect(screen.getByText('invoke-child')).toBeTruthy();
    expect(screen.getByText('01INVOKER01')).toBeTruthy();
    expect(screen.getByText('invoker-fn')).toBeTruthy();
  });

  it('does not render Invoked by section when invokedFrom is null', () => {
    render(<LinkedRuns runID="run-self" invoked={[]} invokedFrom={null} />);
    expect(screen.queryByText('Invoked by')).toBeNull();
  });

  it('does not render Invoked by section when invokedFrom is undefined', () => {
    render(<LinkedRuns runID="run-self" invoked={[]} />);
    expect(screen.queryByText('Invoked by')).toBeNull();
  });

  it("shows '-' for step name when invokedFrom.stepName is null", () => {
    const invokedFrom: RunInvokedFromSummary = {
      parentRunID: '01INVOKER01',
      parentRun: {
        id: '01INVOKER01',
        status: 'COMPLETED',
        function: { name: 'Invoker Fn', slug: 'invoker-fn' },
      },
      stepName: null,
    };
    render(<LinkedRuns runID="run-self" invoked={[]} invokedFrom={invokedFrom} />);
    expect(screen.getByText('Invoked by')).toBeTruthy();
    // Status, run ID, and function pill render — step name cell is '-'.
    const dashes = screen.getAllByText('-');
    expect(dashes.length).toBeGreaterThan(0);
  });

  it('still renders Invoked by row when parent run is null', () => {
    const invokedFrom: RunInvokedFromSummary = {
      parentRunID: '01INVOKER01',
      parentRun: null,
      stepName: 'invoke-child',
    };
    render(<LinkedRuns runID="run-self" invoked={[]} invokedFrom={invokedFrom} />);
    expect(screen.getByText('Invoked by')).toBeTruthy();
    // Only the parent-run link is rendered when parentRun is null.
    const links = screen.getAllByRole('link');
    expect(links).toHaveLength(1);
    expect(links[0]?.getAttribute('href')).toBe('/runs/01INVOKER01');
  });

  it('renders rows inside a semantic table with column headers', () => {
    render(
      <LinkedRuns
        runID="run-self"
        invoked={[]}
        defers={[
          makeDefer({
            userDeferID: 'order-7',
            run: {
              id: '01CHILDRUN01',
              status: 'COMPLETED',
              function: { name: 'Child Fn', slug: 'child-fn' },
            },
          }),
        ]}
      />
    );
    expect(screen.getByRole('table')).toBeTruthy();
    expect(screen.getByRole('columnheader', { name: 'Status' })).toBeTruthy();
    expect(screen.getByRole('columnheader', { name: 'Defer ID' })).toBeTruthy();
    expect(screen.getByRole('columnheader', { name: 'Run ID' })).toBeTruthy();
    expect(screen.getByRole('columnheader', { name: 'Function' })).toBeTruthy();
  });
});
