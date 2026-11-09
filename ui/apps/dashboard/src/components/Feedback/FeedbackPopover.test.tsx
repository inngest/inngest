// @vitest-environment jsdom
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';

import FeedbackPopover from './FeedbackPopover';

const mocks = vi.hoisted(() => ({
  getToken: vi.fn().mockResolvedValue('session-token'),
  success: vi.fn(),
  error: vi.fn(),
}));

vi.mock('@clerk/tanstack-react-start', () => ({
  useAuth: () => ({ getToken: mocks.getToken }),
  useUser: () => ({ user: { id: 'user-1' } }),
  useOrganization: () => ({ organization: { id: 'org-1' } }),
}));
vi.mock('sonner', () => ({
  toast: { success: mocks.success, error: mocks.error },
}));

beforeEach(() => {
  vi.stubEnv('VITE_API_URL', 'https://api.inngest.com');
  const modals = document.createElement('div');
  modals.id = 'modals';
  document.body.appendChild(modals);
  window.history.replaceState(
    null,
    '',
    '/env/production/runs?secret=value#private',
  );
});
afterEach(() => {
  cleanup();
  document.getElementById('modals')?.remove();
  vi.clearAllMocks();
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

const submit = async () => {
  render(<FeedbackPopover trigger={<button>Send feedback</button>} />);
  fireEvent.click(screen.getByRole('button', { name: 'Send feedback' }));
  fireEvent.change(await screen.findByRole('textbox'), {
    target: { value: 'More examples please' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Submit' }));
};

it('sends authenticated feedback through the shared endpoint without client identity or URL secrets', async () => {
  const fetch = vi
    .fn()
    .mockResolvedValue(
      new Response(JSON.stringify({ data: { accepted: true } })),
    );
  vi.stubGlobal('fetch', fetch);
  await submit();
  await waitFor(() => expect(mocks.success).toHaveBeenCalled());
  expect(fetch).toHaveBeenCalledOnce();
  const [url, init] = fetch.mock.calls[0] as [URL, RequestInit];
  expect(url.toString()).toBe('https://api.inngest.com/v2/feedback');
  expect(new Headers(init.headers).get('Authorization')).toBe(
    'Bearer session-token',
  );
  expect(JSON.parse(init.body as string)).toEqual({
    feedback: 'More examples please',
    operation: '/env/production/runs',
  });
  expect(screen.getByRole<HTMLTextAreaElement>('textbox').value).toBe('');
});

it.each([429, 502])(
  'keeps feedback and shows the server error without retrying a %s',
  async (status) => {
    const message =
      status === 429
        ? 'Feedback rate limit exceeded'
        : 'Retrying may create a duplicate ticket';
    const fetch = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ errors: [{ message }] }), { status }),
      );
    vi.stubGlobal('fetch', fetch);
    await submit();
    await waitFor(() => expect(mocks.error).toHaveBeenCalledWith(message));
    expect(fetch).toHaveBeenCalledOnce();
    expect(mocks.success).not.toHaveBeenCalled();
    expect(screen.getByRole<HTMLTextAreaElement>('textbox').value).toBe(
      'More examples please',
    );
  },
);

it('counts Unicode characters and prevents submissions over the limit', async () => {
  const fetch = vi.fn();
  vi.stubGlobal('fetch', fetch);
  render(<FeedbackPopover trigger={<button>Send feedback</button>} />);
  fireEvent.click(screen.getByRole('button', { name: 'Send feedback' }));
  const textbox = await screen.findByRole('textbox');
  fireEvent.change(textbox, { target: { value: '😀'.repeat(10_000) } });
  expect(screen.getByText('10,000 / 10,000 characters')).toBeDefined();
  expect(
    screen.getByRole<HTMLButtonElement>('button', { name: 'Submit' }).disabled,
  ).toBe(false);
  fireEvent.change(textbox, { target: { value: '😀'.repeat(10_001) } });
  expect(screen.getByText('10,001 / 10,000 characters')).toBeDefined();
  expect(
    screen.getByRole<HTMLButtonElement>('button', { name: 'Submit' }).disabled,
  ).toBe(true);
  fireEvent.submit(textbox.closest('form')!);
  expect(fetch).not.toHaveBeenCalled();
});
