import { graphqlAPI } from '@/queries/graphqlAPI';
import {
  hasDeepLinkParams,
  isExpired,
  isValidDeepLink,
  type DashboardDeepLinkSearchParams,
} from '@/lib/deepLinkUtils';
import { auth, clerkClient } from '@clerk/tanstack-react-start/server';
import { createServerFn } from '@tanstack/react-start';
import { createHmac, timingSafeEqual } from 'node:crypto';

export type { DashboardDeepLinkSearchParams } from '@/lib/deepLinkUtils';

type ResolveDashboardDeepLinkInput = DashboardDeepLinkSearchParams & {
  isJWTAuth: boolean;
};

type ResolveDashboardDeepLinkResult =
  | {
      status: 'none';
    }
  | {
      status: 'invalid';
    }
  | {
      status: 'valid';
      shouldSwitchOrganization: boolean;
      organizationId?: string;
    };

export const resolveDashboardDeepLink = createServerFn({
  method: 'GET',
})
  .inputValidator((data: ResolveDashboardDeepLinkInput) => data)
  .handler(async ({ data }) => {
    if (!hasDeepLinkParams(data)) {
      return { status: 'none' } satisfies ResolveDashboardDeepLinkResult;
    }

    if (!isValidDeepLink(data)) {
      return { status: 'invalid' } satisfies ResolveDashboardDeepLinkResult;
    }

    const secret = process.env.AGENTAPI_HMAC_SECRET;
    if (!secret) {
      throw new Error('Missing AGENTAPI_HMAC_SECRET environment variable');
    }

    if (isExpired(data.expires)) {
      return { status: 'invalid' } satisfies ResolveDashboardDeepLinkResult;
    }

    const expectedSignature = signDeepLink(
      secret,
      data.acct,
      data.org,
      data.expires,
    );
    if (!signaturesMatch(expectedSignature, data.sig)) {
      return { status: 'invalid' } satisfies ResolveDashboardDeepLinkResult;
    }

    if (data.isJWTAuth) {
      const result = await graphqlAPI.request<{
        account: {
          id: string;
        };
      }>(`
        query CurrentAccountForDeepLink {
          account {
            id
          }
        }
      `);

      return result.account.id === data.acct
        ? ({
            status: 'valid',
            shouldSwitchOrganization: false,
          } satisfies ResolveDashboardDeepLinkResult)
        : ({ status: 'invalid' } satisfies ResolveDashboardDeepLinkResult);
    }

    const { userId, orgId } = await auth();
    if (!userId) {
      return { status: 'invalid' } satisfies ResolveDashboardDeepLinkResult;
    }

    //
    // Verify the user belongs to the org from the signed deep link.
    // The HMAC proves the org-to-account mapping is correct; this single
    // Clerk call confirms membership instead of paginating all memberships.
    const { data: memberships } =
      await clerkClient().users.getOrganizationMembershipList({
        userId,
        limit: 200,
        offset: 0,
      });
    if (!memberships.some((m) => m.organization.id === data.org)) {
      return { status: 'invalid' } satisfies ResolveDashboardDeepLinkResult;
    }

    return {
      status: 'valid',
      shouldSwitchOrganization: data.org !== orgId,
      organizationId: data.org,
    } satisfies ResolveDashboardDeepLinkResult;
  });

export { hasDeepLinkParams, stripDeepLinkParams } from '@/lib/deepLinkUtils';

function signDeepLink(
  secret: string,
  accountId: string,
  orgId: string,
  expires: string,
): string {
  return createHmac('sha256', secret)
    .update(`${accountId}.${orgId}.${expires}`)
    .digest('hex');
}

function signaturesMatch(expected: string, actual: string): boolean {
  const expectedBuffer = Buffer.from(expected, 'hex');
  const actualBuffer = Buffer.from(actual, 'hex');

  return (
    expectedBuffer.length === actualBuffer.length &&
    timingSafeEqual(expectedBuffer, actualBuffer)
  );
}
