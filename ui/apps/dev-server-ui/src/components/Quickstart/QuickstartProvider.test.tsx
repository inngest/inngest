import { cleanup, render, screen, act } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { QuickstartProvider, useQuickstart } from './QuickstartProvider';
import { Home } from '@/routes/index';
import { storageKey, type Attempt, type QuickstartRun } from './state';

const fixtures = vi.hoisted(() => ({
  pathname: '/',
  info: {} as any,
  apps: {} as any,
  runs: {} as any,
  search: {} as any,
  runsHook: vi.fn(),
  track: vi.fn(),
}));
vi.mock('@/hooks/useTracking', () => ({
  useTracking: () => ({ trackEvent: fixtures.track }),
}));
vi.mock('@/store/devApi', () => ({ useInfoQuery: () => fixtures.info }));
vi.mock('@/store/generated', () => ({
  AppMethod: { Api: 'API' },
  FunctionRunStatus: { Completed: 'COMPLETED' },
  useGetAppsQuery: () => fixtures.apps,
  useQuickstartRunsQuery: (...args: unknown[]) => {
    fixtures.runsHook(...args);
    return { ...fixtures.runs, currentData: fixtures.runs.data };
  },
}));
vi.mock('@tanstack/react-router', () => ({
  createFileRoute: () => () => ({ useSearch: () => fixtures.search }),
  useLocation: () => fixtures.pathname,
  Navigate: () => <div>Runs destination</div>,
}));
vi.mock('@/components/Layout/Layout', () => ({
  default: ({ children }: any) => children,
}));
vi.mock('@/components/Quickstart/Quickstart', () => ({
  Quickstart: ({ showSetupGuide }: { showSetupGuide?: boolean }) => (
    <div>{showSetupGuide ? 'Setup guide' : 'Quickstart screen'}</div>
  ),
}));
vi.mock('@inngest/components/Tooltip', () => ({
  TooltipProvider: ({ children }: any) => children,
}));
const server = 'http://localhost:8288';
const run: QuickstartRun = {
  id: 'run-1',
  appID: 'app-1',
  status: 'COMPLETED',
  queuedAt: '2026-10-23T12:00:01Z',
  endedAt: '2026-10-23T12:00:02Z',
  function: { id: 'fn-1', name: 'Hello', slug: 'app-hello' },
};
const attempt = { functionID: 'fn-1', startedAt: '2026-10-23T12:00:00Z' };
function saveAttempt(overrides: Partial<Attempt> = {}) {
  const value = { ...attempt, ...overrides };
  sessionStorage.setItem(
    `${storageKey(server)}:attempt`,
    JSON.stringify(value),
  );
  return value;
}
function setRuns(...runs: QuickstartRun[]) {
  fixtures.runs.data.runs.edges = runs.map((node) => ({ node }));
}
let state: ReturnType<typeof useQuickstart>;
function Probe() {
  state = useQuickstart();
  return null;
}
function TestApp() {
  return (
    <QuickstartProvider>
      <Probe />
      <Home />
    </QuickstartProvider>
  );
}
beforeEach(() => {
  vi.stubEnv('VITE_PUBLIC_API_BASE_URL', server);
  localStorage.clear();
  sessionStorage.clear();
  fixtures.runsHook.mockClear();
  fixtures.track.mockClear();
  fixtures.search = {};
  fixtures.pathname = '/';
  fixtures.info = { data: { isSingleNodeService: false } };
  fixtures.apps = {
    data: {
      apps: [
        {
          id: 'app-1',
          name: 'App',
          connected: true,
          method: 'SERVE',
          functions: [{ id: 'fn-1', name: 'Hello', slug: 'app-hello' }],
        },
      ],
    },
  };
  fixtures.runs = {
    isSuccess: true,
    data: { runs: { edges: [], pageInfo: { hasNextPage: false } } },
  };
});
afterEach(() => {
  cleanup();
  vi.unstubAllEnvs();
});
it('sends returning users with a retained successful run directly to Runs', () => {
  setRuns(run);
  render(<TestApp />);
  expect(screen.getByText('Runs destination')).toBeTruthy();
  expect(state.completion?.runID).toBe(run.id);
  expect(fixtures.track).not.toHaveBeenCalled();
});
it('keeps the current success visible but routes to Runs on the next entry', () => {
  const view = render(<TestApp />);
  expect(screen.getByText('Quickstart screen')).toBeTruthy();
  setRuns(run);
  view.rerender(<TestApp />);
  expect(screen.getByText('Quickstart screen')).toBeTruthy();
  expect(state.completion?.runID).toBe(run.id);
  view.unmount();
  render(<TestApp />);
  expect(screen.getByText('Runs destination')).toBeTruthy();
});
it('allows explicit reopening without deleting the completion marker', () => {
  fixtures.search = { quickstart: true };
  localStorage.setItem(
    storageKey(server),
    JSON.stringify({
      version: 1,
      server,
      completedAt: run.endedAt,
      appID: run.appID,
      functionID: run.function.id,
      runID: run.id,
    }),
  );
  render(<TestApp />);
  expect(screen.getByText('Setup guide')).toBeTruthy();
  expect(state.completion?.runID).toBe(run.id);
});
it('restores an in-flight attempt without invoking again or completing on failure', () => {
  saveAttempt();
  setRuns({ ...run, status: 'FAILED' });
  render(<TestApp />);
  expect(state.attempt).toEqual(attempt);
  expect(state.completion).toBeNull();
  expect(state.run?.status).toBe('FAILED');
  expect(fixtures.track).not.toHaveBeenCalled();
  expect(fixtures.runsHook).toHaveBeenLastCalledWith(
    expect.objectContaining({
      functionIDs: ['fn-1'],
      from: attempt.startedAt,
      status: null,
    }),
    expect.anything(),
  );
});
it('excludes disconnected apps and durable endpoints from historical inference', () => {
  fixtures.apps.data.apps.push(
    {
      id: 'app-api',
      connected: true,
      method: 'API',
      functions: [{ id: 'fn-api' }],
    },
    { id: 'app-offline', connected: false, functions: [{ id: 'fn-offline' }] },
  );
  render(<TestApp />);
  expect(state.functions.map((fn) => fn.id)).toEqual(['fn-1']);
});
it('preserves serve mode entry and skips onboarding run queries', () => {
  fixtures.info.data.isSingleNodeService = true;
  render(<TestApp />);
  expect(screen.getByText('Runs destination')).toBeTruthy();
  expect(fixtures.runsHook).toHaveBeenLastCalledWith(
    expect.anything(),
    expect.objectContaining({ skip: true }),
  );
});
it('shows the error recovery screen when apps cannot be loaded', () => {
  fixtures.apps = { isError: true };
  render(<TestApp />);
  expect(screen.getByText('Quickstart screen')).toBeTruthy();
  expect(state.ready).toBe(true);
});

it('observes agent runs after prompt handoff and ignores older successful runs', () => {
  fixtures.apps.data.apps = [];
  saveAttempt({ mode: 'agent', functionID: '' });
  const view = render(<TestApp />);
  expect(state.ready).toBe(true);
  fixtures.apps.data.apps = [
    {
      id: 'app-1',
      connected: true,
      method: 'SERVE',
      functions: [{ id: 'fn-1', name: 'Hello', slug: 'app-hello' }],
    },
  ];
  setRuns({ ...run, queuedAt: '2026-10-23T11:00:00Z' });
  view.rerender(<TestApp />);
  expect(state.completion).toBeNull();
  expect(state.attempt?.functionID).toBe('fn-1');
  expect(fixtures.runsHook).toHaveBeenLastCalledWith(
    expect.objectContaining({
      functionIDs: ['fn-1'],
      from: attempt.startedAt,
      status: null,
    }),
    expect.anything(),
  );
  setRuns(run);
  view.rerender(<TestApp />);
  expect(state.completion?.runID).toBe(run.id);
  expect(fixtures.track).toHaveBeenCalledWith('cli/onboarding.action', {
    flow: 'quickstart',
    version: 1,
    action: 'first_run_completed',
    path: 'agent',
  });
  view.rerender(<TestApp />);
  expect(fixtures.track).toHaveBeenCalledTimes(1);
});
it('requires a function choice for agent handoff when multiple functions register', () => {
  saveAttempt({ mode: 'agent', functionID: '' });
  fixtures.apps.data.apps[0].functions.push({
    id: 'fn-2',
    name: 'Other',
    slug: 'app-other',
  });
  setRuns(run);
  render(<TestApp />);
  expect(state.ready).toBe(true);
  expect(state.completion).toBeNull();
  expect(fixtures.runsHook).toHaveBeenLastCalledWith(
    expect.objectContaining({ functionIDs: [] }),
    expect.objectContaining({ skip: true }),
  );
});

it('reopens the guide when a historical completion is found before its marker is saved', () => {
  fixtures.search = { quickstart: true };
  setRuns(run);
  render(<TestApp />);
  expect(screen.getByText('Setup guide')).toBeTruthy();
  expect(state.completion?.runID).toBe(run.id);
});

it('observes another invocation from the reopened guide without clearing completion or counting another conversion', () => {
  fixtures.search = { quickstart: true };
  setRuns(run);
  render(<TestApp />);
  act(() =>
    state.setAttempt({ functionID: 'fn-1', startedAt: '2026-10-27T12:00:00Z' }),
  );
  expect(fixtures.runsHook).toHaveBeenLastCalledWith(
    expect.objectContaining({ from: '2026-10-27T12:00:00Z', status: null }),
    expect.objectContaining({ skip: false, pollingInterval: 1500 }),
  );
  expect(state.completion?.runID).toBe(run.id);
  expect(fixtures.track).not.toHaveBeenCalled();
});

it.each(['FAILED', 'CANCELLED', 'SKIPPED'])(
  'releases a %s attempt after the run handoff so reopening quickstart can retry',
  (status) => {
    saveAttempt();
    setRuns({ ...run, status });
    const view = render(<TestApp />);
    expect(state.attempt).toEqual(attempt);
    expect(state.run?.id).toBe(run.id);
    fixtures.pathname = '/runs';
    view.rerender(<TestApp />);
    expect(state.attempt).toBeNull();
    expect(sessionStorage.getItem(`${storageKey(server)}:attempt`)).toBeNull();
    expect(state.completion).toBeNull();
    expect(fixtures.track).not.toHaveBeenCalled();
    fixtures.pathname = '/';
    view.rerender(<TestApp />);
    expect(state.run).toBeUndefined();
    expect(screen.getByText('Quickstart screen')).toBeTruthy();
  },
);

it.each(['QUEUED', 'RUNNING'])(
  'keeps observing a %s attempt on Runs',
  (status) => {
    saveAttempt();
    fixtures.pathname = '/runs';
    setRuns({ ...run, status });
    render(<TestApp />);
    expect(state.attempt).toEqual(attempt);
    expect(state.completion).toBeNull();
  },
);

it.each(['COMPLETED', 'FAILED', 'CANCELLED', 'SKIPPED'])(
  'clears a later %s attempt while retaining the original completion',
  (status) => {
    fixtures.search = { quickstart: true };
    setRuns(run);
    const view = render(<TestApp />);
    const original = localStorage.getItem(storageKey(server));
    const startedAt = '2026-10-27T12:00:00Z';
    act(() => state.setAttempt({ functionID: 'fn-1', startedAt }));
    setRuns({ ...run, id: 'later-run', status, queuedAt: startedAt });
    view.rerender(<TestApp />);
    expect(state.run?.id).toBe('later-run');
    expect(state.attempt).not.toBeNull();
    fixtures.pathname = '/runs';
    view.rerender(<TestApp />);
    expect(state.attempt).toBeNull();
    expect(state.completion?.runID).toBe(run.id);
    expect(localStorage.getItem(storageKey(server))).toBe(original);
    expect(fixtures.track).not.toHaveBeenCalled();
  },
);
