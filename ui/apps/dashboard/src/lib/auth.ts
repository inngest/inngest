import { auth } from '@clerk/tanstack-react-start/server';
import { redirect } from '@tanstack/react-router';
import { createServerFn } from '@tanstack/react-start';
import { getCookies } from '@tanstack/react-start/server';

export const fetchClerkAuth = createServerFn({ method: 'GET' })
  .inputValidator((data: { redirectUrl?: string }) => data)
  .handler(async ({ data: { redirectUrl } }) => {
    const { isAuthenticated, userId, getToken } = await auth();

    if (!isAuthenticated) {
      throw redirect({
        to: '/sign-in/$',
        search: { redirect_url: redirectUrl },
      });
    }

    const token = await getToken();
    return {
      userId,
      token,
    };
  });

export const jwtAuth = createServerFn({ method: 'GET' }).handler(async () =>
  Object.keys(getCookies()).some((cookie: string) => {
    // Our non-Clerk JWT is either named "jwt" or "jwt-staging".
    return cookie.startsWith('jwt');
  }),
);
