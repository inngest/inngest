/**
 * StepInfo component tests.
 * Feature: exe-1233, Task: 001-retry-attempt-badge
 *
 * Tests are written FIRST per TDD approach.
 */

import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';

import { TooltipProvider } from '../Tooltip/Tooltip';
import { StepInfo } from './StepInfo';
import type { Trace } from './types';

// jsdom doesn't provide ResizeObserver
beforeAll(() => {
  global.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

// Mock modules that use self-referencing @inngest/components/* imports
// which cannot resolve in vitest without a resolve alias.
vi.mock('../Button/Button', () => ({
  Button: (props: Record<string, unknown>) => (
    <button data-testid="mock-button">{props.label as string}</button>
  ),
}));

vi.mock('../Pill/Pill', () => ({
  Pill: ({ children, className }: { children: React.ReactNode; className?: string }) => (
    <span data-testid="mock-pill" className={className}>
      {children}
    </span>
  ),
}));

vi.mock('../DetailsCard/Element', () => ({
  CodeElement: ({ value }: { value: string }) => <code>{value}</code>,
  ElementWrapper: ({ label, children }: { label: string; children: React.ReactNode }) => (
    <div data-label={label}>{children}</div>
  ),
  LinkElement: ({ children }: { children: React.ReactNode }) => <a>{children}</a>,
  TextElement: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
  TimeElement: ({ date }: { date: Date }) => <time>{date.toISOString()}</time>,
}));

vi.mock('../Time', () => ({
  Time: ({ value }: { value: Date }) => <time>{value.toISOString()}</time>,
}));

vi.mock('../AI/AITrace', () => ({
  AITrace: () => null,
}));

vi.mock('../Rerun/RerunModal', () => ({
  RerunModal: () => null,
}));

vi.mock('./ErrorInfo', () => ({
  ErrorInfo: () => null,
}));

vi.mock('./IO', () => ({
  IO: () => null,
}));

vi.mock('./MetadataAttrs', () => ({
  MetadataAttrs: () => null,
}));

vi.mock('./UserlandAttrs', () => ({
  UserlandAttrs: () => null,
}));

vi.mock('./Tabs', () => ({
  Tabs: () => null,
}));

// Mock dependencies
vi.mock('../SharedContext/SharedContext', () => ({
  useShared: () => ({ cloud: false }),
}));

vi.mock('../SharedContext/useBooleanFlag', () => ({
  useBooleanFlag: () => ({
    booleanFlag: () => ({ value: false, isReady: true }),
  }),
}));

vi.mock('../SharedContext/useGetTraceResult', () => ({
  useGetTraceResult: () => ({ loading: false, data: null }),
}));

vi.mock('../SharedContext/usePathCreator', () => ({
  usePathCreator: () => ({
    pathCreator: {
      runPopout: () => '/run/123',
      function: () => '/fn/test',
      eventPopout: () => '/event/123',
    },
  }),
}));

afterEach(() => {
  cleanup();
});

function makeTrace(overrides: Partial<Trace> = {}): Trace {
  return {
    attempts: 0,
    endedAt: null,
    isRoot: false,
    name: 'test-step',
    outputID: null,
    queuedAt: '2026-01-01T00:00:00Z',
    spanID: 'span-1',
    startedAt: '2026-01-01T00:00:01Z',
    status: 'COMPLETED',
    stepInfo: null,
    userlandSpan: null,
    isUserland: false,
    ...overrides,
  };
}

function renderStepInfo(trace: Trace) {
  return render(
    <TooltipProvider>
      <StepInfo selectedStep={{ trace, runID: 'run-1' }} />
    </TooltipProvider>
  );
}

describe('StepInfo retry attempt badge', () => {
  it('renders "Attempt 3" when trace.attempts = 2', () => {
    const trace = makeTrace({ attempts: 2 });
    renderStepInfo(trace);
    const badge = screen.getByTestId('retry-attempt-badge');
    expect(badge).toBeTruthy();
    expect(badge.textContent).toContain('Attempt 3');
  });

  it('does not render badge when trace.attempts = 0 and status is COMPLETED', () => {
    const trace = makeTrace({ attempts: 0, status: 'COMPLETED' });
    renderStepInfo(trace);
    expect(screen.queryByTestId('retry-attempt-badge')).toBeNull();
  });

  it('renders badge when trace.attempts = 0 and status is FAILED', () => {
    const trace = makeTrace({ attempts: 0, status: 'FAILED' });
    renderStepInfo(trace);
    const badge = screen.getByTestId('retry-attempt-badge');
    expect(badge).toBeTruthy();
    expect(badge.textContent).toContain('Attempt 1');
  });

  it('does not render badge when trace.attempts = null', () => {
    const trace = makeTrace({ attempts: null });
    renderStepInfo(trace);
    expect(screen.queryByTestId('retry-attempt-badge')).toBeNull();
  });

  it('shows correct 1-based attempt numbers', () => {
    // attempts=1 → "Attempt 2"
    const trace1 = makeTrace({ attempts: 1 });
    const { unmount: unmount1 } = renderStepInfo(trace1);
    expect(screen.getByTestId('retry-attempt-badge').textContent).toContain('Attempt 2');
    unmount1();

    // attempts=5 → "Attempt 6"
    const trace5 = makeTrace({ attempts: 5 });
    renderStepInfo(trace5);
    expect(screen.getByTestId('retry-attempt-badge').textContent).toContain('Attempt 6');
  });
});
