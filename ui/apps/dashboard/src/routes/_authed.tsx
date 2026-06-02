import { fetchClerkAuth, jwtAuth } from '@/lib/auth';
import { sanitizeRedirectUrl } from '@/lib/deepLinkUtils';
import {
  createFileRoute,
  notFound,
  Outlet,
  useMatch,
} from '@tanstack/react-router';

import Layout from '@/components/Layout/Layout';
import { navCollapsed } from '@/lib/nav';
import { getProfileDisplay } from '@/queries/server/profile';
import NotFound from '@/components/Error/NotFound';

export const Route = createFileRoute('/_authed')({
  component: Authed,
  notFoundComponent: () => {
    return <NotFound />;
  },
  beforeLoad: async ({ location }) => {
    const isJWTAuth = await jwtAuth();

    //
    // for jwt auth (marketplace) abort clerk check below.
    if (isJWTAuth) {
      return;
    }

    const { userId, token } = await fetchClerkAuth({
      data: {
        redirectUrl: sanitizeRedirectUrl(location.href) ?? location.pathname,
      },
    });

    return {
      userId,
      token,
    };
  },

  loader: async () => {
    const profile = await getProfileDisplay();

    if (!profile) {
      throw notFound({ data: { error: 'Profile not found' } });
    }

    return {
      profile,
      navCollapsed: await navCollapsed(),
    };
  },
});

function Authed() {
  const { navCollapsed, profile } = Route.useLoaderData();
  const activeEnv = useMatch({
    from: '/_authed/env/$envSlug',
    shouldThrow: false,
    select: (match) => match.loaderData?.env,
  });

  return (
    <Layout collapsed={navCollapsed} activeEnv={activeEnv} profile={profile}>
      <Outlet />
    </Layout>
  );
}
