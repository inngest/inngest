'use client';

import { useEffect, useState } from 'react';
import dynamic from 'next/dynamic';

import type { Environment } from '@/utils/environments';
import { Alert } from '../Navigation/Alert';
import { Help } from '../Navigation/Help';
import { Integrations } from '../Navigation/Integrations';
import Logo from '../Navigation/Logo';
import Navigation from '../Navigation/Navigation';
import { Profile, type ProfileType } from '../Navigation/Profile';
import useOnboardingWidget from '../Onboarding/useOnboardingWidget';

// Disable SSR in Onboarding Widget Table, to prevent hydration errors. It requires windows info
const OnboardingWidget = dynamic(() => import('../Navigation/OnboardingWidget'), {
  ssr: false,
});

export default function SideBar({
  collapsed: serverCollapsed,
  activeEnv,
  profile,
}: {
  collapsed: boolean | undefined;
  activeEnv?: Environment;
  profile: ProfileType;
}) {
  const [collapsed, setCollapsed] = useState<boolean>(serverCollapsed ?? false);
  const { isWidgetOpen, showWidget, closeWidget } = useOnboardingWidget();

  useEffect(() => {
    //
    // if the user has not set a pref and they are on mobile, collapse by default
    serverCollapsed === undefined &&
      setCollapsed(
        typeof window !== 'undefined' && window.matchMedia('(max-width: 800px)').matches
      );
  }, []);

  return (
    <nav
      className={`bg-canvasBase border-subtle group
         top-0 flex h-screen flex-col justify-start ${
           collapsed ? 'w-[64px]' : 'w-[224px]'
         }  sticky z-[51] shrink-0 overflow-visible border-r`}
    >
      <Logo collapsed={collapsed} setCollapsed={setCollapsed} />
      <div className="flex grow flex-col justify-between">
        <Navigation collapsed={collapsed} activeEnv={activeEnv} />

        <div className="mx-4">
          {!collapsed && <Alert />}
          {isWidgetOpen && <OnboardingWidget collapsed={collapsed} closeWidget={closeWidget} />}
          <Integrations collapsed={collapsed} />
          <Help collapsed={collapsed} showWidget={showWidget} />
        </div>
        <Profile collapsed={collapsed} profile={profile} />
      </div>
    </nav>
  );
}
