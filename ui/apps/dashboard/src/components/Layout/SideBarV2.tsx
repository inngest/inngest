import { useEffect, useRef, useState } from 'react';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { IconPanelLeftClose } from '@inngest/components/icons/PanelLeftClose';
import { IconPanelLeftOpen } from '@inngest/components/icons/PanelLeftOpen';

import type { Environment } from '@/utils/environments';
import AnnouncementStack from '../NavigationV2/Announcements/AnnouncementStack';
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
        collapsed ? 'w-[60px]' : 'w-[220px]'
      } shrink-0 overflow-visible border-r`}
      ref={navRef}
    >
      <div className="flex min-h-0 grow flex-col justify-between">
        {/* Nav list scrolls on short screens; min-h-0 lets the flex child
            actually overflow instead of pushing the bottom widgets off-screen. */}
        <div className="no-scrollbar min-h-0 flex-1 overflow-y-auto">
          <Navigation collapsed={collapsed} activeEnv={activeEnv} />
        </div>

        <div className="shrink-0 pl-3 pr-3">
          <SeatOverageWidget collapsed={collapsed} />
          {/* The onboarding widget takes priority over marketing announcements. */}
          {isWidgetOpen ? (
            <OnboardingWidget collapsed={collapsed} closeWidget={closeWidget} />
          ) : (
            // Marketing announcements are tall and non-essential — drop them on
            // short viewports so the nav + collapse toggle stay reachable.
            <div className="[@media(max-height:720px)]:hidden">
              <AnnouncementStack collapsed={collapsed} />
            </div>
          )}

          {/* Discreet, icon-only collapse toggle pinned to the sidebar foot.
              Hover-revealed via opacity so it stays out of the way; focus-within
              keeps it keyboard reachable while it's transparent. */}
          <div className="pointer-events-none opacity-0 transition-opacity duration-150 focus-within:pointer-events-auto focus-within:opacity-100 group-hover:pointer-events-auto group-hover:opacity-100">
            <OptionalTooltip
              tooltip={
                <span className="flex items-center gap-1.5">
                  {collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
                  <kbd className="border-subtle flex h-4 items-center gap-0.5 rounded border px-1 font-mono text-[10px] font-medium leading-none">
                    <span className="text-xs">⌘</span>B
                  </kbd>
                </span>
              }
            >
              <button
                type="button"
                onClick={toggleCollapsed}
                aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
                className={`text-muted hover:bg-canvasSubtle hover:text-basis my-0.5 flex h-8 w-8 items-center justify-center rounded ${
                  collapsed ? 'mx-auto' : 'mr-auto'
                }`}
              >
                {collapsed ? (
                  <IconPanelLeftOpen className="h-4 w-4" />
                ) : (
                  <IconPanelLeftClose className="h-4 w-4" />
                )}
              </button>
            </OptionalTooltip>
          </div>
        </div>
      </div>
    </nav>
  );
}
