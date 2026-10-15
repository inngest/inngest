// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react';
import {
  afterAll,
  afterEach,
  beforeAll,
  describe,
  expect,
  it,
  vi,
} from 'vitest';

import { ConcurrencyAlertPill } from './ConcurrencyAlertPill';
import type { ConcurrencyAlert } from './useConcurrencyAlert';

const mocks = vi.hoisted(() => ({ useConcurrencyAlert: vi.fn() }));

vi.mock('./useConcurrencyAlert', () => ({
  useConcurrencyAlert: mocks.useConcurrencyAlert,
}));

vi.mock('../ExecutionLimit/ExecutionLimitPill', () => ({
  ExecutionLimitPill: () => <div>execution limit pill</div>,
}));

vi.mock('@inngest/components/Link', () => ({
  Link: ({
    to,
    children,
    'aria-label': ariaLabel,
  }: React.PropsWithChildren<{ to: string; 'aria-label'?: string }>) => (
    <a href={to} aria-label={ariaLabel}>
      {children}
    </a>
  ),
}));

function alert(overrides: Partial<ConcurrencyAlert>): ConcurrencyAlert {
  return {
    state: 'boosted',
    limit: 300,
    minutesRemaining: 42,
    usedRatio: 0.5,
    linkProps: (surface) => ({ to: `/link/${surface}` }),
    ...overrides,
  };
}

class ResizeObserverMock {
  disconnect() {}
  observe() {}
  unobserve() {}
}

beforeAll(() => vi.stubGlobal('ResizeObserver', ResizeObserverMock));
afterAll(() => vi.unstubAllGlobals());

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('ConcurrencyAlertPill', () => {
  it('shows the boost in place of the execution limit pill', () => {
    mocks.useConcurrencyAlert.mockReturnValue(alert({ state: 'boosted' }));
    render(<ConcurrencyAlertPill />);

    expect(screen.queryByText('execution limit pill')).toBeNull();
    expect(screen.getByText('Concurrency limit boosted by 3x.')).toBeTruthy();
    expect(
      screen.getByRole('link', { name: '42 mins left' }).getAttribute('href'),
    ).toBe('/link/pill');
  });

  it('shows the delay in place of the execution limit pill', () => {
    mocks.useConcurrencyAlert.mockReturnValue(alert({ state: 'limited' }));
    render(<ConcurrencyAlertPill />);

    expect(screen.queryByText('execution limit pill')).toBeNull();
    expect(
      screen.getByText(
        'Runs delayed because your concurrency limit was reached.',
      ),
    ).toBeTruthy();
    expect(
      screen
        .getByRole('link', { name: 'Add concurrency' })
        .getAttribute('href'),
    ).toBe('/link/pill');
  });

  it('falls back to the execution limit pill without a concurrency alert', () => {
    mocks.useConcurrencyAlert.mockReturnValue(null);
    render(<ConcurrencyAlertPill />);

    expect(screen.getByText('execution limit pill')).toBeTruthy();
  });
});
