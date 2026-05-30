import { useEffect, useRef, useState } from 'react';
import { RiContractLeftLine, RiContractRightLine } from '@remixicon/react';

import type { Environment } from '@/utils/environments';
import Navigation from '../Navigation/Navigation';
import SeatOverageWidget from '../SeatOverage/SeatOverageWidget';
import OnboardingWidget from '../Navigation/OnboardingWidget';

export default function SideBar({
  collapsed: serverCollapsed,
  activeEnv,
  isWidgetOpen,
  closeWidget,
}: {
  collapsed: boolean | undefined;
  activeEnv?: Environment;
  isWidgetOpen: boolean;
  closeWidget: () => void;
}) {
  const navRef = useRef<HTMLDivElement>(null);

  const [collapsed, setCollapsed] = useState<boolean>(serverCollapsed ?? false);

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

  const toggleCollapsed = () => {
    const toggled = !collapsed;
    setCollapsed(toggled);

    if (typeof window !== 'undefined') {
      window.cookieStore.set('navCollapsed', toggled ? 'true' : 'false');
      // some downstream things, like charts, may need to redraw themselves
      setTimeout(() => window.dispatchEvent(new Event('navToggle')), 200);
    }
  };

  return (
    <nav
      className={`bg-canvasBase border-subtle group relative flex h-full flex-col justify-start py-3 transition-[width] duration-200 ease-out ${
        collapsed ? 'w-[64px]' : 'w-[200px]'
      } shrink-0 overflow-visible border-r-hairline`}
      ref={navRef}
    >
      {/* Floating collapse toggle on the right edge — hover to reveal. */}
      <button
        type="button"
        onClick={toggleCollapsed}
        aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        className="bg-canvasBase border-subtle shadow-xs absolute right-0 top-6 z-[70] hidden h-6 w-6 -translate-y-1/2 translate-x-1/2 items-center justify-center rounded-full border-hairline group-hover:flex"
      >
        {collapsed ? (
          <RiContractRightLine className="text-muted h-3.5 w-3.5" />
        ) : (
          <RiContractLeftLine className="text-muted h-3.5 w-3.5" />
        )}
      </button>

      <div className="flex grow flex-col justify-between">
        <Navigation collapsed={collapsed} activeEnv={activeEnv} />

        <div className="pl-3 pr-2">
          <SeatOverageWidget collapsed={collapsed} />
          {isWidgetOpen && (
            <OnboardingWidget collapsed={collapsed} closeWidget={closeWidget} />
          )}
        </div>
      </div>
    </nav>
  );
}
