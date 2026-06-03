import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { RunDeferSummary, RunDeferredFromSummary } from '../SharedContext/useGetRunLinkage';
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
  LinkCell: ({ href, children }: { href?: string; children: React.ReactNode }) => (
    <a href={href}>{children}</a>
  ),
  StatusCell: ({ status, label }: { status: string; label?: string }) => (
    <span>{label || status}</span>
  ),
  PillCell: ({ href, children }: { href?: string; children: React.ReactNode }) =>
    href ? <a href={href}>{children}</a> : <span>{children}</span>,
}));

vi.mock('../Tooltip/OptionalTooltip', () => ({
  OptionalTooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

afterEach(() => {
  cleanup();
});

function makeDefer(overrides: Partial<RunDeferSummary> = {}): RunDeferSummary {
  return {
    hashedDeferID: 'hash-1',
    userlandDeferID: 'user-id-1',
    fnSlug: 'child-fn',
    status: 'SCHEDULED',
    function: null,
    run: null,
    ...overrides,
  };
}

describe('LinkedRuns', () => {
  it('renders Deferred section for a primary run (no deferredFrom)', () => {
    render(<LinkedRuns defers={[makeDefer()]} />);
    expect(screen.getByText('Deferred runs')).toBeTruthy();
    expect(screen.queryByText('Parent run')).toBeNull();
    expect(screen.queryByText('Parallel defers')).toBeNull();
  });

  it('skips empty Deferred section for a primary run', () => {
    render(<LinkedRuns />);
    expect(screen.queryByText('Deferred runs')).toBeNull();
  });

  it('renders Parent section and skips empty Parallel defers for a deferred run', () => {
    const deferredFrom: RunDeferredFromSummary[] = [
      {
        runID: '01PARENT01',
        function: { name: 'Parent Fn', slug: 'parent-fn' },
        run: null,
      },
    ];
    render(<LinkedRuns deferredFrom={deferredFrom} />);
    expect(screen.getByText('Parent run')).toBeTruthy();
    expect(screen.queryByText('Parallel defers')).toBeNull();
    expect(screen.queryByText('Deferred runs')).toBeNull();
    expect(screen.queryByText('Invoked runs')).toBeNull();
  });

  it('renders the function pill even when the parent run is null', () => {
    const deferredFrom: RunDeferredFromSummary[] = [
      {
        runID: '01PARENT01',
        function: { name: 'Parent Fn', slug: 'parent-fn' },
        run: null,
      },
    ];
    render(<LinkedRuns deferredFrom={deferredFrom} />);
    expect(screen.getByText('Parent Fn')).toBeTruthy();
    // Two links: run-ID link and function link.
    const links = screen.getAllByRole('link');
    expect(links.map((l) => l.getAttribute('href'))).toEqual([
      '/runs/01PARENT01',
      '/functions/parent-fn',
    ]);
  });

  it('renders the parent row without a function link when function is null', () => {
    const deferredFrom: RunDeferredFromSummary[] = [
      {
        runID: '01PARENT01',
        function: null,
        run: null,
      },
    ];
    render(<LinkedRuns deferredFrom={deferredFrom} />);
    // Run ID link is the only link; no /functions/* href is generated when
    // the parent's function couldn't be resolved.
    const links = screen.getAllByRole('link');
    expect(links.map((l) => l.getAttribute('href'))).toEqual(['/runs/01PARENT01']);
    expect(screen.queryByText('Parent Fn')).toBeNull();
  });

  it('parallel defers are passed in directly and exclude the current run', () => {
    const sibling = makeDefer({
      hashedDeferID: 'hash-sibling',
      userlandDeferID: 'user-sibling',
      function: { name: 'Sibling Fn', slug: 'sibling-fn' },
      run: {
        id: 'run-sibling',
        status: 'COMPLETED',
      },
    });
    // The server returns `siblingDefers` already filtered to exclude this run.
    render(<LinkedRuns siblingDefers={[sibling]} />);
    expect(screen.getByText('Parallel defers')).toBeTruthy();
    expect(screen.getByText('user-sibling')).toBeTruthy();
  });

  it('renders a row per parent and the supplied parallel defers for a batched child', () => {
    const siblingA = makeDefer({
      hashedDeferID: 'hash-sibling-a',
      userlandDeferID: 'user-sibling-a',
      function: { name: 'Sibling A', slug: 'sibling-a' },
      run: { id: 'run-sibling-a', status: 'COMPLETED' },
    });
    const siblingB = makeDefer({
      hashedDeferID: 'hash-sibling-b',
      userlandDeferID: 'user-sibling-b',
      function: { name: 'Sibling B', slug: 'sibling-b' },
      run: { id: 'run-sibling-b', status: 'COMPLETED' },
    });
    const deferredFrom: RunDeferredFromSummary[] = [
      {
        runID: '01PARENTA0',
        function: { name: 'Parent A', slug: 'parent-a' },
        run: { id: '01PARENTA0', status: 'COMPLETED' },
      },
      {
        runID: '01PARENTB0',
        function: { name: 'Parent B', slug: 'parent-b' },
        run: { id: '01PARENTB0', status: 'COMPLETED' },
      },
    ];
    render(<LinkedRuns deferredFrom={deferredFrom} siblingDefers={[siblingA, siblingB]} />);

    expect(screen.getByText('Parent runs')).toBeTruthy();
    expect(screen.getByText('01PARENTA0')).toBeTruthy();
    expect(screen.getByText('01PARENTB0')).toBeTruthy();

    // Parallel defers come straight from siblingDefers — the server is
    // responsible for filtering the current run out.
    expect(screen.getByText('user-sibling-a')).toBeTruthy();
    expect(screen.getByText('user-sibling-b')).toBeTruthy();
  });

  it('renders the userlandDeferID, not the hashed id', () => {
    render(
      <LinkedRuns
        defers={[makeDefer({ hashedDeferID: 'sha1-hashed-id', userlandDeferID: 'order-7' })]}
      />
    );
    expect(screen.getByText('order-7')).toBeTruthy();
    expect(screen.queryByText('sha1-hashed-id')).toBeNull();
  });

  it('falls back to fnSlug for the function pill when function is null', () => {
    render(<LinkedRuns defers={[makeDefer({ function: null, fnSlug: 'fallback-fn' })]} />);
    expect(screen.getByText('fallback-fn')).toBeTruthy();
  });

  it("shows '-' in the run-ID column when the run is null", () => {
    render(<LinkedRuns defers={[makeDefer({ run: null })]} />);
    // We only render a '-' for the missing run cell. Status and other cells are
    // present too but `-` should appear at least once.
    const dashes = screen.getAllByText('-');
    expect(dashes.length).toBeGreaterThan(0);
  });

  it('prefers the run status over the defer-row status when a run is linked', () => {
    render(
      <LinkedRuns
        defers={[
          makeDefer({
            status: 'SCHEDULED',
            function: { name: 'Child Fn', slug: 'child-fn' },
            run: {
              id: '01CHILDRUN01',
              status: 'COMPLETED',
            },
          }),
        ]}
      />
    );
    expect(screen.getByText('COMPLETED')).toBeTruthy();
    expect(screen.queryByText('SCHEDULED')).toBeNull();
  });
});
