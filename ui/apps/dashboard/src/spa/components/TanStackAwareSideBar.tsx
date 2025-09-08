import { useEffect, useRef, useState } from 'react';
import dynamic from 'next/dynamic';

import { Help } from '@/components/Navigation/Help';
import { Integrations } from '@/components/Navigation/Integrations';
import Logo from '@/components/Navigation/Logo';
import { Profile } from '@/components/Navigation/Profile';
import useOnboardingWidget from '@/components/Onboarding/useOnboardingWidget';
import type { ProfileDisplayType } from '@/queries/server-only/profile';
import type { Environment } from '@/utils/environments';
import TanStackAwareNavigation from './TanStackAwareNavigation';

// Disable SSR in Onboarding Widget, to prevent hydration errors. It requires windows info
const OnboardingWidget = dynamic(() => import('@/components/Navigation/OnboardingWidget'), {
  ssr: false,
});
const SeatOverageWidget = dynamic(() => import('@/components/SeatOverage/SeatOverageWidget'), {
  ssr: false,
});

export default function TanStackAwareSideBar({
  collapsed: serverCollapsed,
  activeEnv,
  profile,
}: {
  collapsed: boolean | undefined;
  activeEnv?: Environment;
  profile: ProfileDisplayType;
}) {
  const navRef = useRef<HTMLDivElement>(null);

  const [collapsed, setCollapsed] = useState<boolean>(serverCollapsed ?? false);
  const { isWidgetOpen, showWidget, closeWidget } = useOnboardingWidget();

  const autoCollapse = () =>
    typeof window !== 'undefined' &&
    window.matchMedia('(max-width: 800px)').matches &&
    setCollapsed(true);

  useEffect(() => {
    serverCollapsed === undefined && autoCollapse();

    if (navRef.current !== null) {
      window.addEventListener('resize', autoCollapse);

      return () => {
        window.removeEventListener('resize', autoCollapse);
      };
    }
  }, []);

  return (
    <nav
      className={`bg-canvasBase border-subtle group
         top-0 flex h-screen flex-col justify-start ${
           collapsed ? 'w-[64px]' : 'w-[224px]'
         }  sticky z-[51] shrink-0 overflow-visible border-r`}
      ref={navRef}
    >
      <Logo
        collapsed={collapsed}
        envSlug={activeEnv?.slug ?? 'production'}
        envName={activeEnv?.name ?? 'Production'}
        setCollapsed={setCollapsed}
      />
      <div className="flex grow flex-col justify-between">
        <TanStackAwareNavigation collapsed={collapsed} activeEnv={activeEnv} />

        <div className="mx-4">
          <SeatOverageWidget collapsed={collapsed} />
          {isWidgetOpen && <OnboardingWidget collapsed={collapsed} closeWidget={closeWidget} />}
          <Integrations collapsed={collapsed} />
          <Help collapsed={collapsed} showWidget={showWidget} />
        </div>
        <Profile collapsed={collapsed} profile={profile} />
      </div>
    </nav>
  );
}
