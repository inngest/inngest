import { Suspense, type ReactNode } from 'react';

import { type ProfileDisplayType } from '@/queries/server/profile';
import type { Environment } from '@/utils/environments';
import BottomBar from './BottomBar';
import SideBar from './SideBar';
import { ActiveBanners } from '../ActiveBanners/ActiveBanners';
import IncidentBanner from '../Incident/IncidentBanner';

type LayoutProps = {
  collapsed: boolean | undefined;
  activeEnv?: Environment;
  profile?: ProfileDisplayType;
  children: ReactNode;
};

export default function Layout({
  collapsed,
  activeEnv,
  profile,
  children,
}: LayoutProps) {
  return (
    <div className="bg-canvasSubtle flex h-screen w-full flex-col overflow-hidden">
      {/* TODO Phase 4: top bar (org switcher, env switcher, search, avatar). */}
      <div className="h-3 shrink-0" />

      <div
        id="layout-scroll-container"
        className="border-subtle bg-canvasBase mx-3 flex flex-1 flex-row overflow-hidden rounded border"
      >
        <SideBar
          activeEnv={activeEnv}
          collapsed={collapsed}
          profile={profile}
        />

        <div className="no-scrollbar flex flex-1 flex-col overflow-y-auto overflow-x-scroll">
          <IncidentBanner />

          <ActiveBanners />

          {/* disabled by Dan 11/22/2025 for performance reasons */}
          <Suspense>{/* <BillingBanner /> */}</Suspense>
          <Suspense>{/* <ExecutionOverageBanner /> */}</Suspense>

          {children}
        </div>
      </div>

      <BottomBar />
    </div>
  );
}
