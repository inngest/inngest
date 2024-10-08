import {
  auth,
  clerkClient,
  currentUser,
  type Organization,
  type OrganizationMembership,
  type User,
} from '@clerk/nextjs/server';

export type ProfileType = {
  user: User;
  org?: Organization;
};

export type ProfileDisplayType = {
  orgName?: string;
  displayName: string;
};

export const getProfileDisplay = async (): Promise<ProfileDisplayType> => {
  const { user, org } = await getProfile();

  const displayName =
    user.firstName || user.lastName
      ? `${user.firstName ?? ''} ${user.lastName ?? ''}`.trim()
      : user.username ?? '';

  return {
    orgName: org?.name,
    displayName,
  };
};

export const getProfile = async (): Promise<ProfileType> => {
  const user = await currentUser();

  if (!user) {
    throw new Error('User is not logged in');
  }

  const { orgId } = auth();
  return { user, org: orgId ? await getOrg(orgId) : undefined };
};

export const getOrg = async (organizationId: string): Promise<Organization | undefined> => {
  if (!organizationId) {
    return undefined;
  }

  const orgs = (
    await clerkClient().organizations.getOrganizationMembershipList({
      organizationId,
    })
  ).data.map((o: OrganizationMembership) => o.organization);

  return orgs.find((o: Organization) => o.id === organizationId);
};
