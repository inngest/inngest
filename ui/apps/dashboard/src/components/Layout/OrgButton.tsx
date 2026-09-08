import type { ProfileDisplayType } from '@/queries/server/profile';
import OrgAvatar from './OrgAvatar';

export default function OrgButton({
  profile,
}: {
  profile: ProfileDisplayType;
}) {
  const orgName = profile.orgName ?? '';

  return (
    <>
      <OrgAvatar profile={profile} size="sm" />
      <span className="max-w-[140px] truncate leading-normal" title={orgName}>
        {orgName}
      </span>
    </>
  );
}
