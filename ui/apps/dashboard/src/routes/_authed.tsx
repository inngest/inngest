import { fetchClerkAuth, jwtAuth } from '@/lib/auth';
import { sanitizeRedirectUrl } from '@/lib/deepLinkUtils';
import { createFileRoute, notFound, Outlet } from '@tanstack/react-router';

import LayoutV1 from '@/components/Layout/Layout';
import LayoutV2 from '@/components/Layout/LayoutV2';
import { useNavigationV2 } from '@/components/Layout/useNavigationV2';
import { navCollapsed } from '@/lib/nav';
import { getEnvironment } from '@/queries/server/getEnvironment';
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

  loader: async ({ params }: { params: { envSlug?: string } }) => {
    const env = params.envSlug
      ? await getEnvironment({ data: { environmentSlug: params.envSlug } })
      : undefined;

    if (params.envSlug && !env) {
      throw notFound({ data: { error: 'Environment not found' } });
    }

    const profile = await getProfileDisplay();

    if (!profile) {
      throw notFound({ data: { error: 'Profile not found' } });
    }

    return {
      env,
      profile,
      navCollapsed: await navCollapsed(),
    };
  },
});

function Authed() {
  const { env, navCollapsed, profile } = Route.useLoaderData();
  const Layout = useNavigationV2() ? LayoutV2 : LayoutV1;

  return (
    <Layout collapsed={navCollapsed} activeEnv={env} profile={profile}>
      <Outlet />
    </Layout>
  );
}
