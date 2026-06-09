import { Image } from '@unpic/react';

import type { ProfileDisplayType } from '@/queries/server/profile';
import { ProfileMenu } from '../NavigationV2/ProfileMenu';

export default function AvatarMenu({
  profile,
}: {
  profile: ProfileDisplayType;
}) {
  const initial = profile.displayName?.substring(0, 1) || '?';

  return (
    <ProfileMenu isMarketplace={profile.isMarketplace}>
      {profile.userProfilePic ? (
        <Image
          src={profile.userProfilePic}
          className="h-7 w-7 rounded-full object-cover"
          width={28}
          height={28}
          alt="profile-pic"
        />
      ) : (
        initial
      )}
    </ProfileMenu>
  );
}
