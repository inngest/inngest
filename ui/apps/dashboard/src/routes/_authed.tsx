import { fetchClerkAuth, jwtAuth } from '@/lib/auth';
import { createFileRoute, notFound, Outlet } from '@tanstack/react-router';

import Layout from '@/components/Layout/Layout';
import { navCollapsed } from '@/lib/nav';
import { getEnvironment } from '@/queries/server/getEnvironment';
import { getProfileDisplay } from '@/queries/server/profile';

export const Route = createFileRoute('/_authed')({
  component: Authed,
  beforeLoad: async ({ location }) => {
    const isJWTAuth = await jwtAuth();

    //
    // for jwt auth (marketplace) abort clerk check below.
    if (isJWTAuth) {
      return;
    }

    const { userId, token } = await fetchClerkAuth({
      data: { redirectUrl: location.href },
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
