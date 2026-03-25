import { graphqlAPI } from '@/queries/graphqlAPI';
import {
  hasDeepLinkParams,
  isExpired,
  isValidDeepLink,
  type DashboardDeepLinkSearchParams,
} from '@/lib/deepLinkUtils';
import {
  auth,
  clerkClient,
  type OrganizationMembership,
} from '@clerk/tanstack-react-start/server';
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

    const expectedSignature = signDeepLink(secret, data.acct, data.expires);
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

    const user = await clerkClient().users.getUser(userId);
    const memberships = await clerkClient().users.getOrganizationMembershipList(
      {
        userId: user.id,
        limit: 100,
      },
    );

    const membership = memberships.data.find((membership) =>
      organizationMatchesAccount(membership, data.acct),
    );

    if (!membership) {
      return { status: 'invalid' } satisfies ResolveDashboardDeepLinkResult;
    }

    return {
      status: 'valid',
      shouldSwitchOrganization: membership.organization.id !== orgId,
      organizationId: membership.organization.id,
    } satisfies ResolveDashboardDeepLinkResult;
  });

export { hasDeepLinkParams, stripDeepLinkParams } from '@/lib/deepLinkUtils';

function signDeepLink(
  secret: string,
  accountId: string,
  expires: string,
): string {
  return createHmac('sha256', secret)
    .update(`${accountId}.${expires}`)
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

function organizationMatchesAccount(
  membership: OrganizationMembership,
  accountId: string,
): boolean {
  return membership.organization.publicMetadata?.accountID === accountId;
}
