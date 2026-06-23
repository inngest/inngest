import { Suspense, type ReactNode } from 'react';
import { useLocation } from '@tanstack/react-router';

import { type ProfileDisplayType } from '@/queries/server/profile';
import type { Environment } from '@/utils/environments';
import BottomBar from './BottomBar';
import SideBar from './SideBar';
import TopBar from './TopBar';
import { ActiveBanners } from '../ActiveBanners/ActiveBanners';
import IncidentBanner from '../Incident/IncidentBanner';
import { PaymentStatusBanner } from '../PaymentStatusBanner/PaymentStatusBanner';
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

  // Org-level surfaces (settings, billing) and the All Environments list
  // aren't environment-scoped, so the env navigation in the sidebar is
  // meaningless there. Hide it and let the content fill the width; the org
  // menu in the top bar handles org nav. Match /env exactly so env-scoped
  // pages (/env/<slug>/...) keep the sidebar.
  const { pathname } = useLocation();
  const hideSidebar =
    pathname.startsWith('/settings') ||
    pathname.startsWith('/billing') ||
    pathname === '/env' ||
    pathname === '/env/';

  return (
    <div className="bg-canvasSubtle fixed inset-0 flex flex-col overflow-hidden overscroll-none">
      <TopBar
        activeEnv={activeEnv}
        profile={profile}
        showOnboardingWidget={showWidget}
      />

      <div className="border-subtle bg-canvasBase shadow-xs mx-3 flex flex-1 flex-row overflow-hidden rounded-lg border">
        {!hideSidebar && (
          <SideBar
            activeEnv={activeEnv}
            collapsed={collapsed}
            isWidgetOpen={isWidgetOpen}
            closeWidget={closeWidget}
          />
        )}

        <div
          id="layout-scroll-container"
          className="no-scrollbar flex flex-1 flex-col overflow-auto overscroll-contain"
        >
          <IncidentBanner />

          <ActiveBanners />

          <PaymentStatusBanner />

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
