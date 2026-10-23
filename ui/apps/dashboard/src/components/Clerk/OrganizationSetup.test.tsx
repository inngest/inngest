// @vitest-environment jsdom
import { cleanup, render } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';

import { Route } from '@/routes/(auth)/organization-setup';

const mocks = vi.hoisted(() => ({
  request: vi.fn(),
  track: vi.fn(),
  redirect: vi.fn(),
  context: { setUpAccountID: 'account-id' as string | null },
}));

vi.mock('@tanstack/react-router', () => ({
  createFileRoute: () => (config: object) => ({
    options: config,
    useRouteContext: () => mocks.context,
  }),
}));
vi.mock('@tanstack/react-start', () => ({
  createServerFn: () => {
    const builder = {
      inputValidator: () => builder,
      handler:
        (handler: (input: { data?: string }) => unknown) =>
        (input: { data?: string } = {}) =>
          handler(input),
    };
    return builder;
  },
}));
vi.mock('@tanstack/react-start/server', () => ({
  getRequest: () => new Request('https://app.inngest.com'),
}));
vi.mock('@/queries/graphqlAPI', () => ({
  default: { request: mocks.request },
}));
vi.mock('@/utils/accountCreatedTracking', () => ({
  trackAccountCreated: mocks.track,
  ACCOUNT_CREATED_TIMEOUT_MS: 2000,
}));
vi.mock('@/components/Clerk/ReloadClerkAndRedirect', () => ({
  default: mocks.redirect,
}));

beforeEach(() => {
  vi.clearAllMocks();
  mocks.context.setUpAccountID = 'account-id';
  mocks.redirect.mockReturnValue(null);
});
afterEach(cleanup);

it('sets up the account without consuming the conversion claim', async () => {
  mocks.request.mockResolvedValue({
    setUpAccount: { account: { id: 'account-id' } },
  });
  const beforeLoad = Route.options.beforeLoad as () => Promise<unknown>;
  expect(await beforeLoad()).toEqual({ setUpAccountID: 'account-id' });
  expect(mocks.track).not.toHaveBeenCalled();
  expect(mocks.request).toHaveBeenCalledOnce();
});

it('claims only when the tracker is ready, even for a webhook-created account', async () => {
  const Component = Route.options.component!;
  render(<Component />);
  const user = { primaryEmailAddress: { emailAddress: 'pat@example.com' } };
  await mocks.redirect.mock.calls[0][0].beforeRedirect(user);
  expect(mocks.request).not.toHaveBeenCalled();
  const input = mocks.track.mock.calls[0][0];
  expect(input.accountID).toBe('account-id');
  expect(input.email).toBe('pat@example.com');

  mocks.request.mockResolvedValue({ claimSignupConversion: true });
  expect(await input.claimConversion(new AbortController().signal)).toBe(true);
  expect(mocks.request.mock.calls[0][0].variables).toEqual({
    accountID: 'account-id',
  });

  mocks.request.mockResolvedValue({ claimSignupConversion: false });
  expect(await input.claimConversion(new AbortController().signal)).toBe(false);
});
