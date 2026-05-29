import { Suspense, type ReactNode } from 'react';

import { type ProfileDisplayType } from '@/queries/server/profile';
import type { Environment } from '@/utils/environments';
import BottomBar from './BottomBar';
import SideBar from './SideBar';
import TopBar from './TopBar';
import { ActiveBanners } from '../ActiveBanners/ActiveBanners';
import IncidentBanner from '../Incident/IncidentBanner';
import useOnboardingWidget from '../Onboarding/useOnboardingWidget';

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
  // Hoist the onboarding-widget state so the trigger can live in the org
  // menu (top bar) while the widget itself renders in the sidebar.
  const { isWidgetOpen, showWidget, closeWidget } = useOnboardingWidget();

  return (
    <div className="bg-canvasSubtle flex h-screen w-full flex-col overflow-hidden">
      <TopBar
        activeEnv={activeEnv}
        profile={profile}
        showOnboardingWidget={showWidget}
      />

      <div className="border-subtle bg-canvasBase shadow-xs mx-3 flex flex-1 flex-row overflow-hidden rounded border-hairline">
        <SideBar
          activeEnv={activeEnv}
          collapsed={collapsed}
          isWidgetOpen={isWidgetOpen}
          closeWidget={closeWidget}
        />

        <div
          id="layout-scroll-container"
          className="no-scrollbar flex flex-1 flex-col overflow-auto"
        >
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
