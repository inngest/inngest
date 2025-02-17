'use client';

import { useEffect, useRef, useState } from 'react';
import dynamic from 'next/dynamic';

import type { ProfileDisplayType } from '@/queries/server-only/profile';
import type { Environment } from '@/utils/environments';
import { Help } from '../Navigation/Help';
import { Integrations } from '../Navigation/Integrations';
import Logo from '../Navigation/Logo';
import Navigation from '../Navigation/Navigation';
import { Profile } from '../Navigation/Profile';
import useOnboardingWidget from '../Onboarding/useOnboardingWidget';

// Disable SSR in Onboarding Widget, to prevent hydration errors. It requires windows info
const OnboardingWidget = dynamic(() => import('../Navigation/OnboardingWidget'), {
  ssr: false,
});

export default function SideBar({
  collapsed: serverCollapsed,
  activeEnv,
  enableQuickSearchV2,
  profile,
}: {
  collapsed: boolean | undefined;
  activeEnv?: Environment;
  enableQuickSearchV2: boolean;
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
    //
    // if the user has not set a pref and they are on mobile, collapse by default
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
        enableQuickSearchV2={enableQuickSearchV2}
        envSlug={activeEnv?.slug ?? 'production'}
        setCollapsed={setCollapsed}
      />
      <div className="flex grow flex-col justify-between">
        <Navigation collapsed={collapsed} activeEnv={activeEnv} />

        <div className="mx-4">
          {isWidgetOpen && <OnboardingWidget collapsed={collapsed} closeWidget={closeWidget} />}
          <Integrations collapsed={collapsed} />
          <Help collapsed={collapsed} showWidget={showWidget} />
        </div>
        <Profile collapsed={collapsed} profile={profile} />
      </div>
    </nav>
  );
}
