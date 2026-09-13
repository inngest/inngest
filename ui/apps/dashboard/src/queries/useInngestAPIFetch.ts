import { useMemo } from 'react';
import { useAuth } from '@clerk/tanstack-react-start';

type GetSessionToken = (options?: {
  skipCache?: boolean;
}) => Promise<string | null>;

export type InngestAPIFetch = (
  pathname: string,
  init?: RequestInit,
) => Promise<Response>;

export function createInngestAPIFetch(
  getToken: GetSessionToken,
  environmentSlug?: string,
): InngestAPIFetch {
  return async (pathname, init = {}) => {
    const headers = new Headers(init.headers);
    const sessionToken = await getToken({ skipCache: true });
    if (sessionToken) {
      headers.set('Authorization', `Bearer ${sessionToken}`);
    }
    if (environmentSlug) {
      headers.set('X-Inngest-Env', environmentSlug);
    }

    return fetch(new URL(pathname, import.meta.env.VITE_API_URL), {
      ...init,
      // Non-Clerk sessions (e.g. marketplace) authenticate with a non-Clerk
      // JWT cookie. Clerk sessions use the bearer token above.
      credentials: 'include',
      headers,
    });
  };
}

export function useInngestAPIFetch(environmentSlug?: string): InngestAPIFetch {
  const { getToken } = useAuth();

  return useMemo(
    () => createInngestAPIFetch(getToken, environmentSlug),
    [environmentSlug, getToken],
  );
}
