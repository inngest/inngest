import { Image } from '@unpic/react';

import type { ProfileDisplayType } from '@/queries/server/profile';
import { ProfileMenu } from '../Navigation/ProfileMenu';

export default function AvatarMenu({
  profile,
}: {
  profile: ProfileDisplayType;
}) {
  const initial = profile.displayName?.substring(0, 1) || '?';

  return (
    <ProfileMenu isMarketplace={profile.isMarketplace}>
      <div className="bg-canvasMuted text-subtle flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-full text-xs uppercase">
        {profile.orgProfilePic ? (
          <Image
            src={profile.orgProfilePic}
            className="h-8 w-8 rounded-full object-cover"
            width={32}
            height={32}
            alt="profile-pic"
          />
        ) : (
          initial
        )}
      </div>
    </ProfileMenu>
  );
}
