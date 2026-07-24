import { fetchClerkAuth, jwtAuth } from '@/lib/auth';
import { sanitizeRedirectUrl } from '@/lib/deepLinkUtils';
import {
  createFileRoute,
  notFound,
  Outlet,
  useMatch,
} from '@tanstack/react-router';

import LayoutV1 from '@/components/Layout/Layout';
import LayoutV2 from '@/components/Layout/LayoutV2';
import { useNavigationV2 } from '@/components/Layout/useNavigationV2';
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
  const Layout = useNavigationV2() ? LayoutV2 : LayoutV1;
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
