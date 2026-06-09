import { useEffect, useRef, useState } from 'react';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { RiArrowLeftSLine, RiArrowRightSLine } from '@remixicon/react';

import type { Environment } from '@/utils/environments';
import Navigation from '../NavigationV2/Navigation';
import OnboardingWidget from '../NavigationV2/OnboardingWidget';
import SeatOverageWidget from '../SeatOverage/SeatOverageWidget';

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

  // ⌘ B / Ctrl + B toggles the sidebar, except when the user is typing in a
  // form field — otherwise typing "b" with a modifier in the search bar would
  // collapse the sidebar from under them.
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key !== 'b' || !(e.metaKey || e.ctrlKey)) return;
      const el = document.activeElement as HTMLElement | null;
      if (
        el &&
        (el.tagName === 'INPUT' ||
          el.tagName === 'TEXTAREA' ||
          el.isContentEditable)
      ) {
        return;
      }
      e.preventDefault();
      toggleCollapsed();
    }

    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, [collapsed]);

  return (
    <nav
      className={`bg-canvasBase border-subtle group relative flex h-full flex-col justify-start py-3 transition-[width] duration-200 ease-out ${
        collapsed ? 'w-[60px]' : 'w-[200px]'
      } shrink-0 overflow-visible border-r-hairline`}
      ref={navRef}
    >
      {/* Vertical tab on the divider, hover-revealed. Sits half inside / half
          outside the sidebar so it reads as part of the right edge itself. */}
      <OptionalTooltip
        tooltip={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
      >
        <button
          type="button"
          onClick={toggleCollapsed}
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          className="bg-canvasBase border-subtle text-muted hover:text-basis shadow-xs absolute right-0 top-1/2 z-[70] hidden h-8 w-5 -translate-y-1/2 translate-x-1/2 items-center justify-center rounded-md border-hairline transition-colors group-hover:flex"
        >
          {collapsed ? (
            <RiArrowRightSLine className="h-4 w-4" />
          ) : (
            <RiArrowLeftSLine className="h-4 w-4" />
          )}
        </button>
      </OptionalTooltip>

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
