'use client';

import { useEffect, useState, type ReactNode } from 'react';

import SideBar from './SideBar';

export default function Layout({ children }: { children: ReactNode }) {
  const [collapsed, setCollapsed] = useState<boolean | undefined>(undefined);

  useEffect(() => {
    setCollapsed(localStorage.getItem('navCollapsed') === 'true');
  });

  useEffect(() => {
    collapsed !== undefined && localStorage.setItem('navCollapsed', JSON.stringify(collapsed));
  }, [collapsed]);

  //
  // don't render until we have the nav collapsed state to avoid jank
  return (
    <div className="no-scrollbar text-basis fixed z-50 flex h-screen w-full flex-row justify-start overflow-y-scroll overscroll-y-none">
      {collapsed === undefined ? null : (
        <>
          <SideBar collapsed={collapsed} setCollapsed={setCollapsed} />
          <div className="no-scrollbar flex w-full flex-col overflow-x-scroll">{children}</div>
        </>
      )}
    </div>
  );
}
