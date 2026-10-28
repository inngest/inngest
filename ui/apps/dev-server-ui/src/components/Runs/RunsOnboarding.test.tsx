import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { RunsOnboarding } from './RunsOnboarding';

const fixtures = vi.hoisted(() => ({ state: {} as any, track: vi.fn() }));
vi.mock('@/components/Quickstart/QuickstartProvider', () => ({
  useQuickstart: () => fixtures.state,
}));
vi.mock('@/components/Quickstart/useQuickstartTracking', () => ({
  useQuickstartTracking: () => fixtures.track,
}));
vi.mock('@/components/Quickstart/FirstRun', () => ({
  FirstRun: () => <div>Existing function invocation</div>,
}));
beforeEach(() => {
  fixtures.track.mockClear();
  fixtures.state = {
    ready: true,
    isDevServer: true,
    completion: null,
    functions: [{ id: 'fn-1' }],
    server: 'http://localhost:8288',
    info: {},
  };
});
afterEach(cleanup);
const content = <div>Existing runs list and details</div>;

it('focuses a synced app on invocation and keeps the full Quickstart accessible', () => {
  render(<RunsOnboarding>{content}</RunsOnboarding>);
  expect(screen.getByText('Existing function invocation')).toBeTruthy();
  expect(screen.queryByText('Existing runs list and details')).toBeNull();
  expect(
    screen.getByRole('link', { name: 'Open Quickstart' }).getAttribute('href'),
  ).toBe('/?quickstart=true');
  expect(
    screen.getByRole('link', { name: 'View all runs' }).getAttribute('href'),
  ).toBe('/runs?showAllRuns=true');
  expect(fixtures.track.mock.calls).toEqual([
    ['viewed', { entry_stage: 'run' }, true],
    ['app_ready', {}, true],
  ]);
});

it.each([
  { reason: 'completed onboarding', state: { completion: { runID: 'done' } } },
  { reason: 'no eligible function', state: { functions: [] } },
  { reason: 'single-node mode', state: { isDevServer: false } },
])('retains the existing Runs UI for $reason', ({ state }) => {
  Object.assign(fixtures.state, state);
  render(<RunsOnboarding>{content}</RunsOnboarding>);
  expect(screen.getByText('Existing runs list and details')).toBeTruthy();
  expect(screen.queryByText('Existing function invocation')).toBeNull();
  expect(fixtures.track).not.toHaveBeenCalled();
});

it('shows an identified run immediately, even before onboarding finishes loading', () => {
  fixtures.state.ready = false;
  render(<RunsOnboarding runID="identified-run">{content}</RunsOnboarding>);
  expect(screen.getByText('Existing runs list and details')).toBeTruthy();
  expect(fixtures.track).not.toHaveBeenCalled();
});

it('waits for completion reconciliation before replacing the runs list', () => {
  fixtures.state.ready = false;
  const view = render(<RunsOnboarding>{content}</RunsOnboarding>);
  expect(screen.getByRole('status').textContent).toBe('Loading runs…');
  expect(screen.queryByText('Existing function invocation')).toBeNull();
  fixtures.state.ready = true;
  fixtures.state.completion = { runID: 'historical-success' };
  view.rerender(<RunsOnboarding>{content}</RunsOnboarding>);
  expect(screen.getByText('Existing runs list and details')).toBeTruthy();
  expect(fixtures.track).not.toHaveBeenCalled();
});

it.each([true, false])(
  'keeps the full runs list accessible during recovery with ready=%s',
  (ready) => {
    fixtures.state.ready = ready;
    fixtures.state.attempt = {
      functionID: 'fn-1',
      startedAt: '2026-10-23T12:00:00Z',
    };
    render(<RunsOnboarding showAllRuns>{content}</RunsOnboarding>);
    expect(screen.getByText('Existing runs list and details')).toBeTruthy();
    expect(screen.queryByText('Existing function invocation')).toBeNull();
    expect(screen.queryByRole('status')).toBeNull();
    expect(fixtures.track).not.toHaveBeenCalled();
  },
);
