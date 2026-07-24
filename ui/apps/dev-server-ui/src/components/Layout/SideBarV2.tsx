import { useEffect, useRef } from 'react';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { IconPanelLeftClose } from '@inngest/components/icons/PanelLeftClose';
import { IconPanelLeftOpen } from '@inngest/components/icons/PanelLeftOpen';

import Navigation from '../NavigationV2/Navigation';

type SideBarProps = {
  collapsed: boolean;
  setCollapsed: (collapsed: boolean) => void;
  // Whether the user has an explicit stored preference; when absent we
  // auto-collapse on small viewports.
  hasStoredPref: boolean;
};

export default function SideBarV2({
  collapsed,
  setCollapsed,
  hasStoredPref,
}: SideBarProps) {
  const navRef = useRef<HTMLDivElement>(null);

  const autoCollapse = () =>
    typeof window !== 'undefined' &&
    window.matchMedia('(max-width: 800px)').matches &&
    setCollapsed(true);

  useEffect(() => {
    //
    // if the user has not set a pref and they are on mobile, collapse by default
    !hasStoredPref && autoCollapse();

    window.addEventListener('resize', autoCollapse);

    return () => {
      window.removeEventListener('resize', autoCollapse);
    };
  }, [hasStoredPref]);

  const toggleCollapsed = () => {
    setCollapsed(!collapsed);

    if (typeof window !== 'undefined') {
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
      <div className="flex grow flex-col justify-between">
        <Navigation collapsed={collapsed} />

        <div className="pl-3 pr-2">
          {/* Discreet, icon-only collapse toggle pinned to the sidebar foot.
              Hover-revealed via opacity so it stays out of the way; focus-within
              keeps it keyboard reachable while it's transparent. */}
          <div className="pointer-events-none opacity-0 transition-opacity duration-150 focus-within:pointer-events-auto focus-within:opacity-100 group-hover:pointer-events-auto group-hover:opacity-100">
            <OptionalTooltip
              tooltip={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
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
