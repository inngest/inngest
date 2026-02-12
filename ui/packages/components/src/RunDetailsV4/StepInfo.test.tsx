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
import type { StepInfoWait, Trace } from './types';

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
  LinkElement: ({ children, href }: { children: React.ReactNode; href?: string }) => (
    <a href={href}>{children}</a>
  ),
  IDElement: ({ children }: { children: React.ReactNode }) => (
    <span data-testid="mock-id">{children}</span>
  ),
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
      runPopout: ({ runID }: { runID: string }) => `/runs/${runID}`,
      function: ({ functionSlug }: { functionSlug: string }) => `/functions/${functionSlug}`,
      eventPopout: ({ eventID }: { eventID: string }) => `/events/${eventID}`,
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

function renderStepInfo(trace: Trace, { debug = false }: { debug?: boolean } = {}) {
  return render(
    <TooltipProvider>
      <StepInfo selectedStep={{ trace, runID: 'run-1' }} debug={debug} />
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

describe('InvokeInfo fields', () => {
  const invokeStepInfo = {
    triggeringEventID: '01ABC123TRIGGER',
    functionID: 'my-app-send-email',
    returnEventID: '01XYZ789RETURN',
    runID: 'run-invoked-1',
    timeout: '2026-06-01T00:00:00Z',
    timedOut: false,
  };

  it('renders Function field as link with correct href', () => {
    const trace = makeTrace({ stepInfo: invokeStepInfo });
    renderStepInfo(trace);
    const wrapper = document.querySelector('[data-label="Function"]');
    expect(wrapper).toBeTruthy();
    const link = wrapper!.querySelector('a');
    expect(link).toBeTruthy();
    expect(link!.textContent).toContain('my-app-send-email');
    expect(link!.getAttribute('href')).toBe('/functions/my-app-send-email');
  });

  it('renders Triggering Event ID field as link with correct href', () => {
    const trace = makeTrace({ stepInfo: invokeStepInfo });
    renderStepInfo(trace);
    const wrapper = document.querySelector('[data-label="Triggering Event ID"]');
    expect(wrapper).toBeTruthy();
    const link = wrapper!.querySelector('a');
    expect(link).toBeTruthy();
    expect(link!.textContent).toContain('01ABC123TRIGGER');
    expect(link!.getAttribute('href')).toBe('/events/01ABC123TRIGGER');
  });

  it('renders Return Event ID field as link when non-null', () => {
    const trace = makeTrace({ stepInfo: invokeStepInfo });
    renderStepInfo(trace);
    const wrapper = document.querySelector('[data-label="Return Event ID"]');
    expect(wrapper).toBeTruthy();
    const link = wrapper!.querySelector('a');
    expect(link).toBeTruthy();
    expect(link!.textContent).toContain('01XYZ789RETURN');
    expect(link!.getAttribute('href')).toBe('/events/01XYZ789RETURN');
  });

  it('hides Return Event ID field when returnEventID is null', () => {
    const trace = makeTrace({
      stepInfo: { ...invokeStepInfo, returnEventID: null },
    });
    renderStepInfo(trace);
    const wrapper = document.querySelector('[data-label="Return Event ID"]');
    expect(wrapper).toBeNull();
  });

  it('renders all expected field labels in correct order', () => {
    const trace = makeTrace({ stepInfo: invokeStepInfo });
    renderStepInfo(trace);
    const labels = [
      'Function',
      'Triggering Event ID',
      'Triggered Run ID',
      'Timeout',
      'Timed out',
      'Return Event ID',
    ];
    for (const label of labels) {
      expect(document.querySelector(`[data-label="${label}"]`)).toBeTruthy();
    }
  });
});

describe('WaitInfo fields', () => {
  const waitStepInfo: StepInfoWait = {
    eventName: 'test/event',
    expression: 'event.data.id == async.data.id',
    timeout: '2026-01-01T00:00:00Z',
    foundEventID: '01MATCHED123',
    timedOut: false,
  };

  it('renders Matched Event ID as link when foundEventID is non-null', () => {
    const trace = makeTrace({ stepInfo: waitStepInfo });
    renderStepInfo(trace);
    const wrapper = document.querySelector('[data-label="Matched Event ID"]');
    expect(wrapper).toBeTruthy();
    const link = wrapper!.querySelector('a');
    expect(link).toBeTruthy();
    expect(link!.textContent).toContain('01MATCHED123');
    expect(link!.getAttribute('href')).toBe('/events/01MATCHED123');
  });

  it('hides Matched Event ID when foundEventID is null', () => {
    const trace = makeTrace({
      stepInfo: { ...waitStepInfo, foundEventID: null },
    });
    renderStepInfo(trace);
    const wrapper = document.querySelector('[data-label="Matched Event ID"]');
    expect(wrapper).toBeNull();
  });

  it('renders existing WaitInfo fields', () => {
    const trace = makeTrace({ stepInfo: waitStepInfo });
    renderStepInfo(trace);
    const labels = ['Event name', 'Timeout', 'Timed out', 'Match expression'];
    for (const label of labels) {
      expect(document.querySelector(`[data-label="${label}"]`)).toBeTruthy();
    }
  });
});

describe('Step operation type', () => {
  it('renders SDK-style label for known stepOp', () => {
    const trace = makeTrace({ stepOp: 'INVOKE' });
    renderStepInfo(trace);
    const wrapper = document.querySelector('[data-label="Step Type"]');
    expect(wrapper).toBeTruthy();
    expect(wrapper!.textContent).toBe('step.invoke');
  });

  it.each([
    ['RUN', 'step.run'],
    ['INVOKE', 'step.invoke'],
    ['SLEEP', 'step.sleep'],
    ['WAIT_FOR_EVENT', 'step.waitForEvent'],
    ['AI_GATEWAY', 'step.ai'],
    ['WAIT_FOR_SIGNAL', 'step.waitForSignal'],
  ])('maps stepOp "%s" to label "%s"', (stepOp, expectedLabel) => {
    const trace = makeTrace({ stepOp });
    const { unmount } = renderStepInfo(trace);
    const wrapper = document.querySelector('[data-label="Step Type"]');
    expect(wrapper).toBeTruthy();
    expect(wrapper!.textContent).toBe(expectedLabel);
    unmount();
  });

  it('renders raw value for unknown stepOp', () => {
    const trace = makeTrace({ stepOp: 'FUTURE_OP' });
    renderStepInfo(trace);
    const wrapper = document.querySelector('[data-label="Step Type"]');
    expect(wrapper).toBeTruthy();
    expect(wrapper!.textContent).toBe('FUTURE_OP');
  });

  it('hides Step Type when stepOp is null', () => {
    const trace = makeTrace({ stepOp: null });
    renderStepInfo(trace);
    expect(document.querySelector('[data-label="Step Type"]')).toBeNull();
  });
});

describe('Debug Run ID', () => {
  it('renders when debug=true and debugRunID is non-null', () => {
    const trace = makeTrace({ debugRunID: '01DEBUGRUNID123' });
    renderStepInfo(trace, { debug: true });
    const wrapper = document.querySelector('[data-label="Debug Run ID"]');
    expect(wrapper).toBeTruthy();
    expect(wrapper!.textContent).toBe('01DEBUGRUNID123');
  });

  it('hidden when debug=false', () => {
    const trace = makeTrace({ debugRunID: '01DEBUGRUNID123' });
    renderStepInfo(trace, { debug: false });
    expect(document.querySelector('[data-label="Debug Run ID"]')).toBeNull();
  });

  it('hidden when debugRunID is null', () => {
    const trace = makeTrace({ debugRunID: null });
    renderStepInfo(trace, { debug: true });
    expect(document.querySelector('[data-label="Debug Run ID"]')).toBeNull();
  });
});
