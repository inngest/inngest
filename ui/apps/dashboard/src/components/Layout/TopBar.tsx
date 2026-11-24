import { Suspense, type ReactNode } from 'react';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { cn } from '@inngest/components/utils/classNames';

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
  children,
  focused = false,
}: {
  activeEnv?: Environment;
  profile?: ProfileDisplayType;
  showOnboardingWidget: () => void;
  children?: ReactNode;
  /** Company/profile header for focused account flows without environment navigation. */
  focused?: boolean;
}) {
  return (
    <header
      className={cn(
        'bg-canvasSubtle relative z-[60] flex shrink-0 items-center justify-between gap-3',
        focused ? 'min-h-[42px] flex-wrap px-3 py-1' : 'h-[48px] px-5',
      )}
    >
      <div className="-ml-1 flex h-8 items-center gap-1">
        {profile && (
          <>
            <OrgMenu
              profile={profile}
              showOnboardingWidget={showOnboardingWidget}
            >
              <OrgButton profile={profile} />
            </OrgMenu>
            {!focused && (
              <span
                className="text-disabled flex h-8 items-center leading-none"
                aria-hidden
              >
                /
              </span>
            )}
          </>
        )}
        {!focused && (
          <Suspense fallback={<Skeleton className="h-8 w-40" />}>
            <Environments activeEnv={activeEnv} collapsed={false} />
          </Suspense>
        )}
      </div>
      {children}
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
