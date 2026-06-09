import { Suspense } from 'react';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';

import type { ProfileDisplayType } from '@/queries/server/profile';
import type { Environment } from '@/utils/environments';
import Environments from '../NavigationV2/Environments';
import AvatarMenu from './AvatarMenu';
import OrgButton from './OrgButton';
import { OrgMenu } from './OrgMenu';
import SearchTrigger from './SearchTrigger';

export default function TopBar({
  activeEnv,
  profile,
  showOnboardingWidget,
}: {
  activeEnv?: Environment;
  profile?: ProfileDisplayType;
  showOnboardingWidget: () => void;
}) {
  return (
    <header className="bg-canvasSubtle relative z-[60] flex h-[42px] shrink-0 items-center justify-between gap-3 px-3">
      <div className="flex items-center gap-1">
        {profile && (
          <>
            <OrgMenu
              profile={profile}
              showOnboardingWidget={showOnboardingWidget}
            >
              <OrgButton profile={profile} />
            </OrgMenu>
            <span className="text-disabled text-sm" aria-hidden>
              /
            </span>
          </>
        )}
        <Suspense fallback={<Skeleton className="h-7 w-40" />}>
          <Environments activeEnv={activeEnv} collapsed={false} />
        </Suspense>
      </div>
      <div className="flex items-center gap-3">
        <SearchTrigger
          envSlug={activeEnv?.slug ?? 'production'}
          envName={activeEnv?.name ?? 'Production'}
        />
        {profile && <AvatarMenu profile={profile} />}
      </div>
    </header>
  );
}
