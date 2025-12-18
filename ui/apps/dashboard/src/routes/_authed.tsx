import { fetchClerkAuth, jwtAuth } from '@/lib/auth';
import {
  createFileRoute,
  notFound,
  Outlet,
  useMatch,
} from '@tanstack/react-router';

import Layout from '@/components/Layout/Layout';
import { navCollapsed } from '@/lib/nav';
import { getEnvironment } from '@/queries/server/getEnvironment';
import { getProfileDisplay } from '@/queries/server/profile';
import { SentryWrappedCatchBoundary } from '@/components/Error/DefaultCatchBoundary';
import NotFound from '@/components/Error/NotFound';

export const Route = createFileRoute('/_authed')({
  component: Authed,
  staleTime: 0,
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
        redirectUrl: location.href,
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

  return (
    <Layout collapsed={navCollapsed} activeEnv={env} profile={profile}>
      <Outlet />
    </Layout>
  );
}

//
// Thin layout wrapper because errors here mean most of the nav stuff can't be loaded
function AuthedErrorComponent(
  props: Parameters<typeof SentryWrappedCatchBoundary>[0],
) {
  const navCollapsed = useMatch({
    from: '__root__',
    select: (state) => state.loaderData?.navCollapsed ?? false,
  });

  return (
    <Layout collapsed={navCollapsed}>
      <SentryWrappedCatchBoundary {...props} />
    </Layout>
  );
}
