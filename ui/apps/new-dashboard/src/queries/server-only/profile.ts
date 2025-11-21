import { graphql } from "@/gql";
import { graphqlAPI } from "../graphqlAPI";
import {
  type Organization,
  type OrganizationMembership,
  type User,
} from "@clerk/tanstack-react-start/server";
import { auth, clerkClient } from "@clerk/tanstack-react-start/server";
import { createServerFn } from "@tanstack/react-start";

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

const ProfileQuery = graphql(`
  query Profile {
    account {
      name
      marketplace
    }
  }
`);

export const getProfileDisplay = createServerFn({
  method: "GET",
}).handler(async () => {
  let orgName: string | undefined;
  let displayName: string;
  let orgProfilePic: string | null;

  const res = await graphqlAPI.request(ProfileQuery);
  if (res.account.marketplace) {
    // Marketplace users are not authed with Clerk.

    orgName = res.account.name ?? undefined;
    displayName = "System";
    orgProfilePic = null;
  } else {
    const { user, org } = await getProfile();
    orgName = org?.name;
    displayName =
      user.firstName || user.lastName
        ? `${user.firstName ?? ""} ${user.lastName ?? ""}`.trim()
        : user.username ?? "";
    orgProfilePic = org?.hasImage ? org.imageUrl : null;
  }

  return {
    isMarketplace: Boolean(res.account.marketplace),
    orgName,
    displayName,
    orgProfilePic,
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
