// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import {
  afterAll,
  afterEach,
  beforeAll,
  describe,
  expect,
  it,
  vi,
} from 'vitest';

import SandboxesEmptyState from './SandboxesEmptyState';
import {
  SandboxesVerificationCallout,
  type SandboxesVerificationState,
} from './SandboxesVerificationCallout';

vi.mock('@/utils/analyticsEvents', () => ({
  trackEmptyStateViewed: vi.fn(),
  trackEmptyStateUpgradeClicked: vi.fn(),
}));

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

const initial: SandboxesVerificationState = {
  status: 'required',
  canVerify: true,
  started: false,
  progress: { completed: 0, total: 2 },
};

function setup(state: SandboxesVerificationState) {
  const onVerify = vi.fn();
  const onRetry = vi.fn();
  const view = render(
    <SandboxesVerificationCallout
      state={state}
      onVerify={onVerify}
      onRetry={onRetry}
    />,
  );
  return { ...view, onVerify, onRetry };
}

describe('SandboxesVerificationCallout', () => {
  it('hands Start to the host without inventing a verification URL', () => {
    const { onVerify, onRetry } = setup(initial);
    const button = screen.getByRole('button', { name: 'Start verification' });
    fireEvent.click(button);
    expect(onVerify).toHaveBeenCalledTimes(1);
    expect(onRetry).not.toHaveBeenCalled();
    expect(screen.queryByRole('link')).toBeNull();
    expect(screen.getByRole('progressbar').getAttribute('aria-valuenow')).toBe(
      '0',
    );
  });

  it('uses supplied progress and hands Continue to the host', () => {
    const { onVerify } = setup({
      ...initial,
      started: true,
      progress: { completed: 1, total: 2 },
    });
    const progress = screen.getByRole('progressbar', {
      name: 'Account verification',
    });
    expect(progress.getAttribute('aria-valuetext')).toBe(
      '1 of 2 checks complete',
    );
    fireEvent.click(
      screen.getByRole('button', { name: 'Continue verification' }),
    );
    expect(onVerify).toHaveBeenCalledTimes(1);
  });

  it('keeps Continue available when a flow has started with no completed checks', () => {
    setup({ ...initial, started: true });
    expect(
      screen.getByRole('button', { name: 'Continue verification' }),
    ).toBeTruthy();
    expect(screen.getByRole('progressbar').getAttribute('aria-valuenow')).toBe(
      '0',
    );
  });

  it('does not hardcode the number of checks', () => {
    setup({ ...initial, started: true, progress: { completed: 2, total: 3 } });
    expect(screen.getByRole('progressbar').getAttribute('aria-valuetext')).toBe(
      '2 of 3 checks complete',
    );
  });

  it('shows only the concise admin treatment to members', () => {
    const { onVerify } = setup({ status: 'required', canVerify: false });
    expect(
      screen.getByRole('heading', { name: 'Ask an admin to verify' }),
    ).toBeTruthy();
    expect(
      screen.getByText('An account admin needs to complete verification.'),
    ).toBeTruthy();
    expect(screen.getByText('Unverified')).toBeTruthy();
    expect(screen.queryByRole('button')).toBeNull();
    expect(screen.queryByRole('progressbar')).toBeNull();
    expect(onVerify).not.toHaveBeenCalled();
  });

  it.each(['verified', 'not-required'] as const)(
    'renders nothing when %s',
    (status) => {
      const { container } = setup({ status });
      expect(container.innerHTML).toBe('');
    },
  );

  it('announces loading without showing an active verification action', () => {
    setup({ status: 'loading' });
    expect(screen.getByRole('status').getAttribute('aria-busy')).toBe('true');
    expect(screen.getByRole('status').textContent).toContain(
      'Loading verification status',
    );
    expect(screen.queryByRole('button')).toBeNull();
    expect(screen.queryByRole('progressbar')).toBeNull();
  });

  it('exposes retry without suggesting that verification is required', () => {
    const { onVerify, onRetry } = setup({ status: 'error' });
    expect(screen.getByRole('alert').textContent).toContain(
      'Unable to load verification',
    );
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(onRetry).toHaveBeenCalledTimes(1);
    expect(onVerify).not.toHaveBeenCalled();
    expect(screen.queryByRole('progressbar')).toBeNull();
  });

  it('removes the callout after the host confirms completion', () => {
    const { rerender, container, onVerify, onRetry } = setup(initial);
    rerender(
      <SandboxesVerificationCallout
        state={{ status: 'verified' }}
        onVerify={onVerify}
        onRetry={onRetry}
      />,
    );
    expect(container.innerHTML).toBe('');
    expect(screen.queryByText('Account verified')).toBeNull();
  });
});

describe('Sandboxes zero-state composition', () => {
  it('preserves the existing zero state when no verification state is supplied', () => {
    render(<SandboxesEmptyState />);
    expect(screen.getByRole('heading', { name: 'Sandboxes' })).toBeTruthy();
    expect(
      screen.getByRole('heading', {
        name: 'Sandboxes are available on paid plans',
      }),
    ).toBeTruthy();
    expect(
      screen.getAllByRole('button', { name: 'Upgrade plan' }),
    ).toHaveLength(2);
    expect(screen.queryByRole('complementary')).toBeNull();
  });

  it.each<SandboxesVerificationState>([
    initial,
    { ...initial, started: true, progress: { completed: 1, total: 2 } },
    { status: 'required', canVerify: false },
    { status: 'loading' },
    { status: 'error' },
    { status: 'verified' },
  ])(
    'keeps only the upgrade treatment when an upgrade is required with status $status',
    (state) => {
      const callbacks = { onVerify: vi.fn(), onRetry: vi.fn() };
      const { container, rerender } = render(<SandboxesEmptyState />);
      const originalContent = container.firstElementChild?.textContent;
      const originalLinks = Array.from(
        container.querySelectorAll('a'),
        (link) => link.getAttribute('href'),
      );
      rerender(
        <SandboxesEmptyState
          requiresUpgrade
          verification={{ state, ...callbacks }}
        />,
      );
      expect(container.firstElementChild?.textContent).toBe(originalContent);
      expect(
        Array.from(container.querySelectorAll('a'), (link) =>
          link.getAttribute('href'),
        ),
      ).toEqual(originalLinks);
      expect(
        screen.getAllByRole('button', { name: 'Upgrade plan' }),
      ).toHaveLength(2);
      expect(screen.queryByRole('complementary')).toBeNull();
      expect(callbacks.onVerify).not.toHaveBeenCalled();
      expect(callbacks.onRetry).not.toHaveBeenCalled();
    },
  );

  it('shows verification instead of upgrades when verification is required and hides the gate after completion', () => {
    const callbacks = { onVerify: vi.fn(), onRetry: vi.fn() };
    const { container, rerender } = render(
      <SandboxesEmptyState requiresUpgrade={false} />,
    );
    const originalContent = container.firstElementChild?.textContent;

    rerender(
      <SandboxesEmptyState
        requiresUpgrade={false}
        verification={{ state: initial, ...callbacks }}
      />,
    );
    expect(screen.getByRole('complementary')).toBeTruthy();
    expect(
      screen.getByRole('heading', { name: 'What can you build?' }),
    ).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Start verification' }));
    expect(callbacks.onVerify).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole('button', { name: 'Upgrade plan' })).toBeNull();
    expect(screen.queryByText('Paid plans')).toBeNull();
    expect(
      screen.queryByRole('heading', {
        name: 'Sandboxes are available on paid plans',
      }),
    ).toBeNull();

    rerender(
      <SandboxesEmptyState
        requiresUpgrade={false}
        verification={{ state: { status: 'verified' }, ...callbacks }}
      />,
    );
    expect(screen.queryByRole('complementary')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Upgrade plan' })).toBeNull();
    expect(container.firstElementChild?.textContent).toBe(originalContent);
    expect(container.firstElementChild?.className).toContain('max-w-[816px]');
  });

  it('clears the prior action and progress when the host supplies a different account state', () => {
    const callbacks = { onVerify: vi.fn(), onRetry: vi.fn() };
    const { rerender } = render(
      <SandboxesEmptyState
        requiresUpgrade={false}
        verification={{
          state: {
            ...initial,
            started: true,
            progress: { completed: 1, total: 2 },
          },
          ...callbacks,
        }}
      />,
    );
    expect(
      screen.getByRole('button', { name: 'Continue verification' }),
    ).toBeTruthy();
    rerender(
      <SandboxesEmptyState
        requiresUpgrade={false}
        verification={{
          state: { status: 'required', canVerify: false },
          ...callbacks,
        }}
      />,
    );
    expect(
      screen.queryByRole('button', { name: 'Continue verification' }),
    ).toBeNull();
    expect(screen.queryByRole('progressbar')).toBeNull();
    expect(
      screen.getByRole('heading', { name: 'Ask an admin to verify' }),
    ).toBeTruthy();
  });

  it('removes stale verification actions when switching to an account that requires an upgrade', () => {
    const callbacks = { onVerify: vi.fn(), onRetry: vi.fn() };
    const verification = { state: initial, ...callbacks };
    const { rerender } = render(
      <SandboxesEmptyState
        requiresUpgrade={false}
        verification={verification}
      />,
    );
    expect(
      screen.getByRole('button', { name: 'Start verification' }),
    ).toBeTruthy();
    rerender(
      <SandboxesEmptyState requiresUpgrade verification={verification} />,
    );
    expect(
      screen.queryByRole('button', { name: 'Start verification' }),
    ).toBeNull();
    expect(screen.queryByRole('complementary')).toBeNull();
    expect(
      screen.getAllByRole('button', { name: 'Upgrade plan' }),
    ).toHaveLength(2);
  });
});
