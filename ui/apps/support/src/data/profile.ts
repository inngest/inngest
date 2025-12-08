// import { graphql } from "@/gql";

import {
  Organization,
  OrganizationMembership,
  User,
} from "@clerk/tanstack-react-start/server";
import { auth, clerkClient } from "@clerk/tanstack-react-start/server";
import { createServerFn } from "@tanstack/react-start";
// import { inngestGQLAPI } from "./gqlApi";

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

// const ProfileQuery = graphql(`
//   query Profile {
//     account {
//       name
//       marketplace
//     }
//   }
// `);

export const getProfileDisplay = createServerFn({
  method: "GET",
}).handler(async (): Promise<ProfileDisplayType> => {
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
  if (!user) {
    throw new Error("User is not logged in");
  }

  if (!user) {
    throw new Error("User is not logged in");
  }

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
