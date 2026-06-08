import { Image } from '@unpic/react';

import type { ProfileDisplayType } from '@/queries/server/profile';
import { ProfileMenu as ProfileMenuV1 } from '../Navigation/ProfileMenu';
import { ProfileMenu as ProfileMenuV2 } from '../NavigationV2/ProfileMenu';
import { useNavigationV2 } from './useNavigationV2';

export default function AvatarMenu({
  profile,
}: {
  profile: ProfileDisplayType;
}) {
  const ProfileMenu = useNavigationV2() ? ProfileMenuV2 : ProfileMenuV1;
  const initial = profile.displayName?.substring(0, 1) || '?';

  return (
    <ProfileMenu isMarketplace={profile.isMarketplace}>
      <div className="bg-canvasMuted text-subtle flex h-7 w-7 shrink-0 items-center justify-center overflow-hidden rounded-full text-xs uppercase">
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
      </div>
    </ProfileMenu>
  );
}
