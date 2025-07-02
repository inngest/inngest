import { auth } from '@clerk/nextjs/server';

import { graphql } from '@/gql';
import graphqlAPI from '../graphqlAPI';

export type ProfileType = {
  username?: string;
  fullName?: string;
  orgName?: string;
  orgHasImage?: boolean;
  orgImageUrl?: string;
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

export const getProfileDisplay = async (): Promise<ProfileDisplayType> => {
  let orgName: string | undefined;
  let displayName: string;
  let orgProfilePic: string | null;

  const res = await graphqlAPI.request(ProfileQuery);
  if (res.account.marketplace) {
    // Marketplace users are not authed with Clerk.

    orgName = res.account.name ?? undefined;
    displayName = 'System';
    orgProfilePic = null;
  } else {
    const {
      username,
      fullName,
      orgName: orgNameFromClaims,
      orgHasImage,
      orgImageUrl,
    } = await getProfile();
    orgName = orgNameFromClaims;
    displayName = fullName ?? username ?? '';
    orgProfilePic = orgHasImage ? orgImageUrl ?? null : null;
  }

  return {
    isMarketplace: Boolean(res.account.marketplace),
    orgName,
    displayName,
    orgProfilePic,
  };
};

export const getProfile = async (): Promise<ProfileType> => {
  const { userId, sessionClaims } = auth();

  if (!userId) {
    throw new Error('User is not logged in');
  }

  return {
    username: sessionClaims.username,
    fullName: sessionClaims.fullName,
    orgName: sessionClaims.orgName,
    orgHasImage: sessionClaims.orgHasImage,
    orgImageUrl: sessionClaims.orgImageUrl,
  };
};
