import { auth, clerkClient, currentUser } from '@clerk/nextjs';
import type { Organization, OrganizationMembership, User } from '@clerk/nextjs/server';

export type ProfileType = {
  user: User & { displayName: string };
  org: Organization | undefined;
};

export const getProfile = async (): Promise<ProfileType> => {
  const user = await currentUser();

  if (!user) {
    throw new Error('User is not logged in');
  }

  const displayName =
    user.firstName || user.lastName
      ? `${user.firstName ?? ''} ${user.lastName ?? ''}`.trim()
      : user.username ?? '';

  const { orgId } = auth();

  return { user: { ...user, displayName }, org: orgId ? await getOrg(orgId) : undefined };
};

export const getOrg = async (organizationId: string): Promise<Organization | undefined> => {
  if (!organizationId) {
    return undefined;
  }

  const orgs = (
    await clerkClient.organizations.getOrganizationMembershipList({
      organizationId,
    })
  ).map((o: OrganizationMembership) => o.organization);

  return orgs.find((o: Organization) => o.id === organizationId);
};
