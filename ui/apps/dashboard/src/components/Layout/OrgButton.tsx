import { Image } from '@unpic/react';

import type { ProfileDisplayType } from '@/queries/server/profile';

export default function OrgButton({
  profile,
}: {
  profile: ProfileDisplayType;
}) {
  const orgName = profile.orgName ?? '';
  const initial = orgName.substring(0, 1) || '?';

  return (
    <>
      <span className="bg-canvasMuted text-subtle flex border border-muted h-7 w-7 shrink-0 items-center justify-center overflow-hidden rounded-full text-xs uppercase">
        {profile.orgProfilePic ? (
          <Image
            src={profile.orgProfilePic}
            className="h-7 w-7 rounded-full object-cover"
            width={24}
            height={24}
            alt="org-profile-pic"
          />
        ) : (
          initial
        )}
      </span>
      <span className="max-w-[140px] truncate leading-normal" title={orgName}>
        {orgName}
      </span>
    </>
  );
}
