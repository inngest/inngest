// @vitest-environment jsdom
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { TooltipProvider } from '@inngest/components/Tooltip';

import { OAuthSessions } from './OAuthSessions';
import type { OAuthSession } from './useOAuthSessions';

const auth = vi.hoisted(() => ({
  getToken: vi.fn().mockResolvedValue('browser-token'),
  userId: 'user-1',
  orgId: 'org-1',
}));
vi.mock('@clerk/tanstack-react-start', () => ({ useAuth: () => auth }));
vi.mock('sonner', () => ({ toast: { success: vi.fn() } }));

const session: OAuthSession = {
  id: 'session-1',
  name: 'Work laptop',
  client_name: 'Inngest CLI',
  environment: { id: 'production', name: 'Production' },
  permissions: ['apps:read:*'],
  created_at: '2026-01-01T00:00:00Z',
  expires_at: '2099-01-01T00:00:00Z',
  revoked_at: null,
};
const response = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status });
const page = (sessions = [session], has_more = false) =>
  response({ sessions, has_more });
let fetchMock: ReturnType<typeof vi.fn>;

function setup() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  const view = render(
    <QueryClientProvider client={client}>
      <TooltipProvider>
        <OAuthSessions />
      </TooltipProvider>
    </QueryClientProvider>,
  );
  return { ...view, client };
}

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  );
  const modals = document.createElement('div');
  modals.id = 'modals';
  document.body.appendChild(modals);
  fetchMock = vi.fn();
  vi.stubGlobal('fetch', fetchMock);
  vi.stubEnv('VITE_API_URL', 'https://api.inngest.test');
  auth.orgId = 'org-1';
  auth.getToken.mockReset().mockResolvedValue('browser-token');
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
  document.getElementById('modals')?.remove();
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe('OAuth sessions', () => {
  it('shows metadata and permissions, cancels without revoking, and refreshes after confirmation', async () => {
    fetchMock
      .mockResolvedValueOnce(page())
      .mockResolvedValueOnce(response({ status: 'revoked' }))
      .mockResolvedValue(
        page([{ ...session, revoked_at: '2026-10-05T00:00:00Z' }]),
      );
    setup();
    fireEvent.click(await screen.findByRole('button', { name: 'Work laptop' }));
    expect(await screen.findByText('OAuth session details')).toBeTruthy();
    expect(screen.getByText('Apps')).toBeTruthy();
    expect(screen.getByText('Read')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Revoke session' }));
    const dialog = await screen.findByRole('alertdialog');
    expect(within(dialog).getByText('Revoke "Work laptop"?')).toBeTruthy();
    fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }));
    expect(fetchMock).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole('button', { name: 'Work laptop' }));
    fireEvent.click(
      await screen.findByRole('button', { name: 'Revoke session' }),
    );
    fireEvent.click(
      within(await screen.findByRole('alertdialog')).getByRole('button', {
        name: 'Revoke',
      }),
    );
    await screen.findByText('Revoked');
    expect(fetchMock.mock.calls[1]?.[0].pathname).toBe(
      '/oauth/sessions/session-1/revoke',
    );
    expect(fetchMock.mock.calls[1]?.[1]).toMatchObject({
      method: 'POST',
      credentials: 'include',
      headers: { Authorization: 'Bearer browser-token' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Work laptop' }));
    expect(
      (
        await screen.findByRole('button', {
          name: 'Revoke session',
        })
      ).hasAttribute('disabled'),
    ).toBe(true);
  });

  it('keeps the confirmation open after a failed revoke and allows retry', async () => {
    fetchMock
      .mockResolvedValueOnce(page())
      .mockResolvedValueOnce(response({ error: 'server_error' }, 500))
      .mockResolvedValueOnce(response({ status: 'revoked' }))
      .mockResolvedValue(
        page([{ ...session, revoked_at: '2026-10-05T00:00:00Z' }]),
      );
    setup();
    fireEvent.click(await screen.findByRole('button', { name: 'Work laptop' }));
    fireEvent.click(
      await screen.findByRole('button', { name: 'Revoke session' }),
    );
    fireEvent.click(
      within(await screen.findByRole('alertdialog')).getByRole('button', {
        name: 'Revoke',
      }),
    );
    expect(
      await screen.findByText(
        'Could not revoke this session. Please try again.',
      ),
    ).toBeTruthy();
    fireEvent.click(
      within(screen.getByRole('alertdialog')).getByRole('button', {
        name: 'Revoke',
      }),
    );
    expect(await screen.findByText('Revoked')).toBeTruthy();
  });

  it.each(['token', 'fetch'])(
    'allows retry after stalled %s acquisition',
    async (stage) => {
      fetchMock.mockResolvedValueOnce(page());
      setup();
      fireEvent.click(
        await screen.findByRole('button', { name: 'Work laptop' }),
      );
      fireEvent.click(
        await screen.findByRole('button', { name: 'Revoke session' }),
      );
      const dialog = await screen.findByRole('alertdialog');
      let finishToken: (token: string) => void = () => {};
      if (stage === 'token') {
        auth.getToken.mockImplementationOnce(
          () =>
            new Promise<string>((resolve) => {
              finishToken = resolve;
            }),
        );
      } else {
        fetchMock.mockImplementationOnce(() => new Promise(() => {}));
      }
      vi.useFakeTimers();
      fireEvent.click(within(dialog).getByRole('button', { name: 'Revoke' }));
      await act(async () => {
        await vi.advanceTimersByTimeAsync(30_000);
      });
      expect(
        screen.getByText('Could not revoke this session. Please try again.'),
      ).toBeTruthy();
      expect(
        within(dialog)
          .getByRole('button', { name: 'Cancel' })
          .hasAttribute('disabled'),
      ).toBe(false);
      if (stage === 'fetch') {
        expect(fetchMock.mock.calls[1]?.[1].signal.aborted).toBe(true);
      } else {
        await act(async () => finishToken('late-token'));
        expect(fetchMock).toHaveBeenCalledTimes(1);
      }
      vi.useRealTimers();
      fetchMock
        .mockResolvedValueOnce(response({ status: 'revoked' }))
        .mockResolvedValue(
          page([{ ...session, revoked_at: '2026-10-05T00:00:00Z' }]),
        );
      fireEvent.click(within(dialog).getByRole('button', { name: 'Revoke' }));
      expect(await screen.findByText('Revoked')).toBeTruthy();
    },
  );

  it('retries failed loads and shows an empty state', async () => {
    fetchMock
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValue(page([]));
    setup();
    fireEvent.click(await screen.findByRole('button', { name: 'Retry' }));
    expect(await screen.findByText(/No OAuth sessions yet/)).toBeTruthy();
  });

  it('paginates and clears the previous selection', async () => {
    fetchMock
      .mockResolvedValueOnce(page([session], true))
      .mockResolvedValueOnce(
        page([
          {
            ...session,
            id: 'session-2',
            name: 'MCP connection',
            environment: null,
          },
        ]),
      );
    setup();
    fireEvent.click(await screen.findByRole('button', { name: 'Next page' }));
    expect(
      await screen.findByRole('button', { name: 'MCP connection' }),
    ).toBeTruthy();
    expect(screen.getByText('All environments')).toBeTruthy();
    expect(screen.queryByText('Work laptop')).toBeNull();
    expect(fetchMock.mock.calls[1]?.[0].searchParams.get('offset')).toBe('20');
  });

  it('isolates cached sessions by organization', async () => {
    fetchMock.mockResolvedValueOnce(page()).mockResolvedValueOnce(page([]));
    const { client, rerender } = setup();
    await screen.findByRole('button', { name: 'Work laptop' });
    auth.orgId = 'org-2';
    rerender(
      <QueryClientProvider client={client}>
        <TooltipProvider>
          <OAuthSessions key="org-2" />
        </TooltipProvider>
      </QueryClientProvider>,
    );
    expect(screen.queryByText('Work laptop')).toBeNull();
    expect(await screen.findByText(/No OAuth sessions yet/)).toBeTruthy();
  });

  it('blocks duplicate revokes and aborts in-flight requests on unmount', async () => {
    let complete: (value: Response) => void = () => {};
    fetchMock.mockResolvedValueOnce(page()).mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          complete = resolve;
        }),
    );
    const { unmount } = setup();
    fireEvent.click(await screen.findByRole('button', { name: 'Work laptop' }));
    fireEvent.click(
      await screen.findByRole('button', { name: 'Revoke session' }),
    );
    const button = within(await screen.findByRole('alertdialog')).getByRole(
      'button',
      { name: 'Revoke' },
    );
    fireEvent.click(button);
    fireEvent.click(button);
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    const signal = fetchMock.mock.calls[1]?.[1].signal as AbortSignal;
    unmount();
    expect(signal.aborted).toBe(true);
    await act(async () => complete(response({ status: 'revoked' })));
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
