import { Image } from '@unpic/react';
import { cn } from '@inngest/components/utils/classNames';

import type { ProfileDisplayType } from '@/queries/server/profile';

// The trigger in the top bar uses a small circle; the org menu header uses a
// larger rounded square.
const sizes = {
  sm: { frame: 'h-7 w-7 rounded-full', text: 'text-xs', px: 28 },
  lg: { frame: 'h-9 w-9 rounded-lg', text: 'text-base', px: 36 },
} as const;

export default function OrgAvatar({
  profile,
  size,
}: {
  profile: ProfileDisplayType;
  size: keyof typeof sizes;
}) {
  const orgName = profile.orgName ?? '';
  const initial = orgName.substring(0, 1) || '?';
  const { frame, text, px } = sizes[size];

  return (
    <span
      className={cn(
        'bg-canvasMuted text-subtle border-muted flex shrink-0 items-center justify-center overflow-hidden border uppercase',
        frame,
        text,
      )}
    >
      {profile.orgProfilePic ? (
        <Image
          src={profile.orgProfilePic}
          className={cn('object-cover', frame)}
          width={px}
          height={px}
          alt="org-profile-pic"
        />
      ) : (
        initial
      )}
    </span>
  );
}
