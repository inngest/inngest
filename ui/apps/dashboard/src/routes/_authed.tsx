import { fetchClerkAuth, jwtAuth } from '@/lib/auth';
import {
  createFileRoute,
  notFound,
  Outlet,
  redirect,
  useMatch,
} from '@tanstack/react-router';

import Layout from '@/components/Layout/Layout';
import { navCollapsed } from '@/lib/nav';
import { getEnvironment } from '@/queries/server/getEnvironment';
import { getProfileDisplay } from '@/queries/server/profile';
import { SentryWrappedCatchBoundary } from '@/components/Error/DefaultCatchBoundary';
import NotFound from '@/components/Error/NotFound';
import { auth, clerkClient } from '@clerk/tanstack-react-start/server';

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
        redirectUrl: location.href,
      },
    });

    //
    // Check setup status similar to Next.js middleware
    const { orgId } = await auth();
    const user = await clerkClient().users.getUser(userId);
    const isUserSetup = !!user.externalId;
    const hasActiveOrganization = !!orgId;

    //
    // User is not set up yet - redirect to user setup
    if (!isUserSetup) {
      throw redirect({
        to: '/user-setup',
      });
    }

    //
    // User is set up but has no active organization - redirect to organization list
    if (
      isUserSetup &&
      !hasActiveOrganization &&
      !location.pathname.startsWith('/organization-list')
    ) {
      throw redirect({
        to: '/organization-list/$',
        params: { _splat: '' },
        search: {
          redirect_url: location.pathname,
        },
      });
    }

    //
    // User has active org - check if org is set up
    if (isUserSetup && hasActiveOrganization) {
      const org = await clerkClient().organizations.getOrganization({
        organizationId: orgId,
      });
      const isOrganizationSetup = !!(
        org.publicMetadata as { accountID?: string }
      ).accountID;

      //
      // Organization not set up yet - redirect to organization setup
      if (!isOrganizationSetup) {
        throw redirect({
          to: '/organization-setup',
        });
      }
    }

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
