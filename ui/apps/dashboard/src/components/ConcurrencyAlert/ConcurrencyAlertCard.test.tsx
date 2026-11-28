// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { ConcurrencyAlertCard } from './ConcurrencyAlertCard';
import type { ConcurrencyAlert } from './useConcurrencyAlert';

const mocks = vi.hoisted(() => ({ useConcurrencyAlert: vi.fn() }));

vi.mock('./useConcurrencyAlert', () => ({
  useConcurrencyAlert: mocks.useConcurrencyAlert,
}));

vi.mock('../ExecutionLimit/ExecutionLimitCard', () => ({
  ExecutionLimitCard: () => <div>execution limit card</div>,
}));

vi.mock('@inngest/components/Link', () => ({
  Link: ({ to, children }: React.PropsWithChildren<{ to: string }>) => (
    <a href={to}>{children}</a>
  ),
}));

vi.mock('@inngest/components/Menu/MenuItem', () => ({
  MenuItem: ({ to, text }: { to: string; text: string }) => (
    <a href={to}>{text}</a>
  ),
}));

function alert(overrides: Partial<ConcurrencyAlert>): ConcurrencyAlert {
  return {
    state: 'boosted',
    limit: 1500,
    minutesRemaining: 42,
    usedRatio: 0.5,
    linkProps: { to: '/link/card', onClick: vi.fn() },
    ...overrides,
  };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('ConcurrencyAlertCard', () => {
  it('shows the boost in place of the execution limit card', () => {
    mocks.useConcurrencyAlert.mockReturnValue(alert({ state: 'boosted' }));
    render(<ConcurrencyAlertCard collapsed={false} />);

    expect(screen.queryByText('execution limit card')).toBeNull();
    expect(screen.getByText('Concurrency boosted!')).toBeTruthy();
    expect(
      screen.getByText(/boosted by 3x\. Add concurrency to avoid delays/),
    ).toBeTruthy();
    expect(screen.getByText('42 mins remaining this month')).toBeTruthy();
    expect(
      screen.getByRole('link', { name: 'View usage' }).getAttribute('href'),
    ).toBe('/link/card');
  });

  it('shows the limit in place of the execution limit card', () => {
    mocks.useConcurrencyAlert.mockReturnValue(alert({ state: 'limited' }));
    render(<ConcurrencyAlertCard collapsed={false} />);

    expect(screen.queryByText('execution limit card')).toBeNull();
    expect(screen.getByText('Concurrency limit reached')).toBeTruthy();
    expect(
      screen.getByText(/Runs delayed due to your account hitting/),
    ).toBeTruthy();
    expect(screen.getByText('1,500/1,500 concurrent steps')).toBeTruthy();
    expect(
      screen
        .getByRole('link', { name: 'Add concurrency' })
        .getAttribute('href'),
    ).toBe('/link/card');
  });

  it('links the collapsed card', () => {
    mocks.useConcurrencyAlert.mockReturnValue(
      alert({
        state: 'limited',
        linkProps: { to: '/link/card-collapsed', onClick: vi.fn() },
      }),
    );
    render(<ConcurrencyAlertCard collapsed />);

    expect(
      screen
        .getByRole('link', { name: 'Add concurrency' })
        .getAttribute('href'),
    ).toBe('/link/card-collapsed');
  });

  it('falls back to the execution limit card without a concurrency alert', () => {
    mocks.useConcurrencyAlert.mockReturnValue(null);
    render(<ConcurrencyAlertCard collapsed={false} />);

    expect(screen.getByText('execution limit card')).toBeTruthy();
  });
});
