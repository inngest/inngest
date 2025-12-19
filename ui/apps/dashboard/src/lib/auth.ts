import { auth, clerkClient } from '@clerk/tanstack-react-start/server';
import { redirect } from '@tanstack/react-router';
import { createServerFn } from '@tanstack/react-start';
import { getCookies } from '@tanstack/react-start/server';

export const fetchClerkAuth = createServerFn({ method: 'GET' })
  .inputValidator((data: { redirectUrl?: string }) => data)
  .handler(async ({ data: { redirectUrl } }) => {
    const { isAuthenticated, userId, getToken, orgId } = await auth();

    if (!isAuthenticated) {
      throw redirect({
        to: '/sign-in/$',
        params: { _splat: '' },
        search: {
          redirect_url: redirectUrl?.startsWith('/') ? redirectUrl : undefined,
        },
      });
    }

    //
    // Check setup status
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
      !redirectUrl?.startsWith('/organization-list')
    ) {
      throw redirect({
        to: '/organization-list/$',
        params: { _splat: '' },
        search: {
          redirect_url: redirectUrl?.startsWith('/') ? redirectUrl : undefined,
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
