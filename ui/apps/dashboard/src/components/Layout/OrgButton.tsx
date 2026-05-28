import { Image } from '@unpic/react';
import { RiArrowDownSLine } from '@remixicon/react';

import type { ProfileDisplayType } from '@/queries/server/profile';

export default function OrgButton({
  profile,
}: {
  profile: ProfileDisplayType;
}) {
  const orgName = profile.orgName ?? '';
  const initial = orgName.substring(0, 1) || '?';

  return (
    <div className="text-basis hover:bg-canvasBase/60 flex items-center gap-2 rounded px-2 py-1 text-sm font-medium">
      <span className="bg-canvasBase text-subtle border-subtle flex h-6 w-6 shrink-0 items-center justify-center overflow-hidden rounded border text-xs uppercase">
        {profile.orgProfilePic ? (
          <Image
            src={profile.orgProfilePic}
            className="h-6 w-6 rounded object-cover"
            width={24}
            height={24}
            alt="org-profile-pic"
          />
        ) : (
          initial
        )}
      </span>
      <span className="max-w-[140px] truncate" title={orgName}>
        {orgName}
      </span>
      <RiArrowDownSLine className="text-muted h-4 w-4 shrink-0" />
    </div>
  );
}
