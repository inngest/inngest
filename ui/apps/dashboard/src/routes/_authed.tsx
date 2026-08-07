import { fetchClerkAuth, jwtAuth } from '@/lib/auth';
import { sanitizeRedirectUrl } from '@/lib/deepLinkUtils';
import {
  createFileRoute,
  notFound,
  Outlet,
  useLocation,
  useMatch,
} from '@tanstack/react-router';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';

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

// Standalone surfaces are approval screens reached from outside the dashboard,
// so the nav would only offer a way to wander off mid-flow.
const STANDALONE_PATHS = ['/device'];

function Authed() {
  const { navCollapsed, profile } = Route.useLoaderData();
  const Layout = useNavigationV2() ? LayoutV2 : LayoutV1;
  const activeEnv = useMatch({
    from: '/_authed/env/$envSlug',
    shouldThrow: false,
    select: (match) => match.loaderData?.env,
  });

  const { pathname } = useLocation();
  if (STANDALONE_PATHS.some((p) => pathname === p || pathname === `${p}/`)) {
    return <StandaloneShell />;
  }

  return (
    <Layout collapsed={navCollapsed} activeEnv={activeEnv} profile={profile}>
      <Outlet />
    </Layout>
  );
}

function StandaloneShell() {
  return (
    <div className="flex h-full flex-col overflow-y-auto">
      <header className="px-6 py-6">
        <InngestLogo />
      </header>
      <div className="mx-auto flex w-full max-w-screen-xl grow items-center px-6">
        <Outlet />
      </div>
    </div>
  );
}
