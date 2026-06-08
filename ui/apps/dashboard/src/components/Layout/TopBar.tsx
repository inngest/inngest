import { Suspense } from 'react';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';

import type { ProfileDisplayType } from '@/queries/server/profile';
import type { Environment } from '@/utils/environments';
import EnvironmentsV1 from '../Navigation/Environments';
import EnvironmentsV2 from '../NavigationV2/Environments';
import AvatarMenu from './AvatarMenu';
import OrgButton from './OrgButton';
import { OrgMenu } from './OrgMenu';
import SearchTrigger from './SearchTrigger';
import { useNavigationV2 } from './useNavigationV2';

export default function TopBar({
  activeEnv,
  profile,
  showOnboardingWidget,
}: {
  activeEnv?: Environment;
  profile?: ProfileDisplayType;
  showOnboardingWidget: () => void;
}) {
  const Environments = useNavigationV2() ? EnvironmentsV2 : EnvironmentsV1;

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
            <span className="text-disabled" aria-hidden>
              /
            </span>
          </>
        )}
        <Suspense fallback={<Skeleton className="h-8 w-40" />}>
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
