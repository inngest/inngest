// import { graphql } from "@/gql";

import { createServerFn } from "@tanstack/react-start";
import { auth, clerkClient } from "@clerk/tanstack-react-start/server";
import type {
  Organization,
  OrganizationMembership,
  User,
} from "@clerk/tanstack-react-start/server";

export type ProfileType = {
  user: User;
  org?: Organization;
};

export type ProfileDisplayType = {
  isMarketplace: boolean;
  orgName?: string;
  displayName: string;
  orgProfilePic: string | null;
};

export const getProfileDisplay = createServerFn({
  method: "GET",
}).handler((): ProfileDisplayType => {
  // TODO: Implement profile display logic
  // For now, return placeholder values
  return {
    isMarketplace: false,
    displayName: "User",
    orgProfilePic: null,
  };
});

export const getProfile = async (): Promise<ProfileType> => {
  const { userId, orgId } = await auth();

  if (!userId) {
    throw new Error("User is not logged in");
  }

  const user = await clerkClient().users.getUser(userId);

  return { user, org: orgId ? await getOrg(orgId) : undefined };
};

export const getOrg = async (
  organizationId: string,
): Promise<Organization | undefined> => {
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
