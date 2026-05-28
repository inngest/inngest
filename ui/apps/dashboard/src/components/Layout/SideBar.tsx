import { useEffect, useRef, useState } from 'react';

import type { Environment } from '@/utils/environments';
import Logo from '../Navigation/Logo';
import Navigation from '../Navigation/Navigation';
import { Integrations } from '../Navigation/Integrations';
import OnboardingGuideTrigger from '../Navigation/OnboardingGuideTrigger';
import useOnboardingWidget from '../Onboarding/useOnboardingWidget';
import SeatOverageWidget from '../SeatOverage/SeatOverageWidget';
import OnboardingWidget from '../Navigation/OnboardingWidget';

export default function SideBar({
  collapsed: serverCollapsed,
  activeEnv,
}: {
  collapsed: boolean | undefined;
  activeEnv?: Environment;
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
  }, [serverCollapsed]);

  return (
    <nav
      className={`bg-canvasBase border-subtle group flex h-full flex-col justify-start ${
        collapsed ? 'w-[64px]' : 'w-[224px]'
      } shrink-0 overflow-visible border-r`}
      ref={navRef}
    >
      <Logo collapsed={collapsed} setCollapsed={setCollapsed} />
      <div className="flex grow flex-col justify-between">
        <Navigation collapsed={collapsed} activeEnv={activeEnv} />

        <div className="mx-4">
          <SeatOverageWidget collapsed={collapsed} />
          {isWidgetOpen ? (
            <OnboardingWidget collapsed={collapsed} closeWidget={closeWidget} />
          ) : (
            <OnboardingGuideTrigger
              collapsed={collapsed}
              showWidget={showWidget}
            />
          )}
          <Integrations collapsed={collapsed} />
        </div>
      </div>
    </nav>
  );
}
