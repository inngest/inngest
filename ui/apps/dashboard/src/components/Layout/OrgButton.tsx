import { Image } from '@unpic/react';
import { RiExpandUpDownLine } from '@remixicon/react';

import type { ProfileDisplayType } from '@/queries/server/profile';

export default function OrgButton({
  profile,
}: {
  profile: ProfileDisplayType;
}) {
  const orgName = profile.orgName ?? '';
  const initial = orgName.substring(0, 1) || '?';

  return (
    <div className="text-basis hover:bg-canvasMuted flex h-8 items-center gap-1.5 rounded px-2 text-sm">
      <span className="bg-canvasMuted text-subtle flex h-7 w-7 shrink-0 items-center justify-center overflow-hidden rounded-full text-xs uppercase">
        {profile.orgProfilePic ? (
          <Image
            src={profile.orgProfilePic}
            className="h-7 w-7 rounded-full object-cover"
            width={28}
            height={28}
            alt="org-profile-pic"
          />
        ) : (
          initial
        )}
      </span>
      <span className="max-w-[140px] truncate" title={orgName}>
        {orgName}
      </span>
      <RiExpandUpDownLine className="text-muted h-4 w-4 shrink-0" />
    </div>
  );
}
