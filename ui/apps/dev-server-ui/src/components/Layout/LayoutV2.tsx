import type { ReactNode } from 'react';

import BottomBar from './BottomBar';
import SideBarV2 from './SideBarV2';
import TopBar from './TopBar';

type LayoutProps = {
  collapsed: boolean;
  setCollapsed: (collapsed: boolean) => void;
  hasStoredPref: boolean;
  children: ReactNode;
};

export default function LayoutV2({
  collapsed,
  setCollapsed,
  hasStoredPref,
  children,
}: LayoutProps) {
  return (
    <div className="bg-canvasSubtle fixed inset-0 flex flex-col overflow-hidden overscroll-none">
      <TopBar />

      <div className="border-subtle bg-canvasBase shadow-xs mx-3 flex flex-1 flex-row overflow-hidden rounded-lg border">
        <SideBarV2
          collapsed={collapsed}
          setCollapsed={setCollapsed}
          hasStoredPref={hasStoredPref}
        />

        <div
          id="layout-scroll-container"
          className="no-scrollbar flex flex-1 flex-col overflow-auto overscroll-contain"
        >
          {children}
        </div>
      </div>

      <BottomBar />
    </div>
  );
}
