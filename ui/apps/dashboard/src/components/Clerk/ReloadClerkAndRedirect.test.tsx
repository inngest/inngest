// @vitest-environment jsdom
import { cleanup, render, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({ useUser: vi.fn() }));

vi.mock('@clerk/tanstack-react-start', () => ({ useUser: mocks.useUser }));
vi.mock('@/components/Icons/LoadingIcon', () => ({ default: () => null }));

import ReloadClerkAndRedirect from './ReloadClerkAndRedirect';

const REDIRECT_URL = '/onboarding';

const replace = vi.fn();
const reloadedUser = { id: 'user_reloaded' };
const reload = vi.fn();

const stubClerk = ({ isLoaded = true }: { isLoaded?: boolean } = {}) => {
  mocks.useUser.mockReturnValue({
    isLoaded,
    user: isLoaded ? { reload } : null,
  });
};

beforeEach(() => {
  reload.mockResolvedValue(reloadedUser);
  vi.stubGlobal('location', { replace });
  stubClerk();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe('ReloadClerkAndRedirect', () => {
  it('does nothing until Clerk has loaded', async () => {
    stubClerk({ isLoaded: false });

    render(<ReloadClerkAndRedirect redirectURL={REDIRECT_URL} />);
    await Promise.resolve();

    expect(reload).not.toHaveBeenCalled();
    expect(replace).not.toHaveBeenCalled();
  });

  it('redirects after reloading the user when there is no beforeRedirect', async () => {
    render(<ReloadClerkAndRedirect redirectURL={REDIRECT_URL} />);

    await waitFor(() => expect(replace).toHaveBeenCalledWith(REDIRECT_URL));
    expect(reload).toHaveBeenCalledTimes(1);
  });

  it('runs beforeRedirect with the reloaded user, then redirects', async () => {
    const order: string[] = [];
    reload.mockImplementation(async () => {
      order.push('reload');
      return reloadedUser;
    });
    const beforeRedirect = vi.fn(async () => {
      order.push('beforeRedirect');
    });
    replace.mockImplementation(() => order.push('redirect'));

    render(
      <ReloadClerkAndRedirect
        redirectURL={REDIRECT_URL}
        beforeRedirect={beforeRedirect}
      />,
    );

    await waitFor(() => expect(replace).toHaveBeenCalledWith(REDIRECT_URL));
    expect(beforeRedirect).toHaveBeenCalledWith(reloadedUser);
    expect(order).toEqual(['reload', 'beforeRedirect', 'redirect']);
  });

  it('waits for beforeRedirect to finish before redirecting', async () => {
    let finish: () => void = () => {};
    const beforeRedirect = vi.fn(
      () => new Promise<void>((resolve) => (finish = resolve)),
    );

    render(
      <ReloadClerkAndRedirect
        redirectURL={REDIRECT_URL}
        beforeRedirect={beforeRedirect}
      />,
    );

    await waitFor(() => expect(beforeRedirect).toHaveBeenCalled());
    expect(replace).not.toHaveBeenCalled();

    finish();

    await waitFor(() => expect(replace).toHaveBeenCalledWith(REDIRECT_URL));
  });

  it('still redirects when beforeRedirect rejects', async () => {
    const beforeRedirect = vi.fn().mockRejectedValue(new Error('gtm blew up'));

    render(
      <ReloadClerkAndRedirect
        redirectURL={REDIRECT_URL}
        beforeRedirect={beforeRedirect}
      />,
    );

    await waitFor(() => expect(replace).toHaveBeenCalledWith(REDIRECT_URL));
    expect(beforeRedirect).toHaveBeenCalledTimes(1);
  });

  it('does not reload again when the callback identity changes', async () => {
    const { rerender } = render(
      <ReloadClerkAndRedirect
        redirectURL={REDIRECT_URL}
        beforeRedirect={async () => {}}
      />,
    );
    await waitFor(() => expect(replace).toHaveBeenCalledTimes(1));

    rerender(
      <ReloadClerkAndRedirect
        redirectURL={REDIRECT_URL}
        beforeRedirect={async () => {}}
      />,
    );
    await Promise.resolve();

    expect(reload).toHaveBeenCalledTimes(1);
    expect(replace).toHaveBeenCalledTimes(1);
  });
});
