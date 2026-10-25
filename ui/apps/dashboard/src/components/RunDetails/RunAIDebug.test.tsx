// @vitest-environment jsdom
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import type { GetRunData } from '@inngest/components/SharedContext/useGetRun';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { RunAIDebug } from './RunAIDebug';

const mocks = vi.hoisted(() => ({
  organization: {
    id: 'org-1',
    name: 'Example',
    publicMetadata: { accountID: 'account-1' },
  } as {
    id: string;
    name: string;
    publicMetadata: { accountID: string };
  } | null,
  environments: {
    data: [{ id: 'env-1', slug: 'default-canonical' }],
    fetching: false,
    error: undefined as Error | undefined,
  },
  refetch: vi.fn(),
  track: vi.fn(),
}));
vi.mock('@clerk/tanstack-react-start', () => ({
  useOrganization: () => ({ organization: mocks.organization }),
}));
vi.mock('@/components/Environments/environment-context', () => ({
  useEnvironment: () => ({ id: 'env-1', slug: 'production' }),
}));
vi.mock('@/queries/environments', () => ({
  useEnvironments: () => [mocks.environments, mocks.refetch],
}));
vi.mock('@/components/Runs/AIAssist/tracking', () => ({
  useRunAITracking: () => mocks.track,
}));

const run = {
  id: 'run-1',
  status: 'FAILED',
  fn: { name: 'Create chat completion' },
} as GetRunData;
beforeEach(() => {
  vi.stubEnv('VITE_API_URL', 'https://api.inngest.com');
  mocks.organization = {
    id: 'org-1',
    name: 'Example',
    publicMetadata: { accountID: 'account-1' },
  };
  mocks.environments = {
    data: [{ id: 'env-1', slug: 'default-canonical' }],
    fetching: false,
    error: undefined,
  };
  const modals = document.createElement('div');
  modals.id = 'modals';
  document.body.appendChild(modals);
});
afterEach(() => {
  cleanup();
  document.getElementById('modals')?.remove();
  vi.clearAllMocks();
  vi.unstubAllEnvs();
});
const open = () =>
  fireEvent.click(screen.getByRole('button', { name: 'Debug with AI' }));

describe('Cloud run debugging', () => {
  it.each([false, true])(
    'copies the exact preview in the detail view (standalone=%s)',
    async (standalone) => {
      const writeText = vi.fn().mockResolvedValue(undefined);
      Object.defineProperty(navigator, 'clipboard', {
        configurable: true,
        value: { writeText },
      });
      const { rerender } = render(
        <RunAIDebug run={run} standalone={standalone} />,
      );
      open();
      const preview = screen.getByRole<HTMLTextAreaElement>('textbox', {
        name: 'Prompt preview',
      });
      expect(preview.readOnly).toBe(true);
      expect(preview.value).toContain('"env": "default-canonical"');
      expect(preview.value).toContain('"runId": "run-1"');
      expect(
        screen.getByText('Example / production · Create chat completion'),
      ).toBeTruthy();
      fireEvent.click(screen.getByRole('button', { name: 'Copy prompt' }));
      await waitFor(() =>
        expect(mocks.track).toHaveBeenCalledWith({ action: 'prompt_copied' }),
      );
      expect(writeText).toHaveBeenCalledWith(preview.value);
      rerender(
        <RunAIDebug run={{ ...run, id: 'run-2' }} standalone={standalone} />,
      );
      expect(screen.queryByRole('dialog')).toBeNull();
      open();
      expect(
        screen.getByRole<HTMLTextAreaElement>('textbox', {
          name: 'Prompt preview',
        }).value,
      ).toContain('"runId": "run-2"');
      expect(screen.queryByRole('button', { name: 'Copied' })).toBeNull();
    },
  );

  it('keeps setup available while unresolved environment metadata blocks copying and supports retry', () => {
    mocks.environments.data = [{ id: 'other-env', slug: 'production' }];
    render(<RunAIDebug run={run} standalone={false} />);
    open();
    expect(
      screen.getByRole<HTMLButtonElement>('button', { name: 'Copy prompt' })
        .disabled,
    ).toBe(true);
    fireEvent.click(screen.getByRole('button', { name: 'Claude Code' }));
    const guide = screen.getByRole<HTMLAnchorElement>('link', {
      name: 'Full setup guide',
    });
    expect(guide.target).toBe('_blank');
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(mocks.refetch).toHaveBeenCalledWith({
      requestPolicy: 'network-only',
    });
  });

  it('blocks a missing stable account identity', () => {
    mocks.organization!.publicMetadata.accountID = '';
    render(<RunAIDebug run={run} standalone={true} />);
    open();
    expect(
      screen.getByRole<HTMLButtonElement>('button', { name: 'Copy prompt' })
        .disabled,
    ).toBe(true);
  });

  it.each(['COMPLETED', 'RUNNING', 'CANCELLED', 'QUEUED'])(
    'hides the entry for %s runs',
    (status) => {
      render(<RunAIDebug run={{ ...run, status }} standalone={false} />);
      expect(
        screen.queryByRole('button', { name: 'Debug with AI' }),
      ).toBeNull();
    },
  );

  it('hides the entry without an authenticated Cloud organization', () => {
    mocks.organization = null;
    render(<RunAIDebug run={run} standalone={false} />);
    expect(screen.queryByRole('button', { name: 'Debug with AI' })).toBeNull();
  });

  it('does not duplicate viewed events during run polling', () => {
    const { rerender } = render(<RunAIDebug run={run} standalone={false} />);
    rerender(<RunAIDebug run={{ ...run }} standalone={false} />);
    expect(
      mocks.track.mock.calls.filter(([event]) => event.action === 'viewed'),
    ).toHaveLength(1);
  });
});
