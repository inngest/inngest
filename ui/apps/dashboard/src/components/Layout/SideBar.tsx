'use client';

import { useEffect, useRef, useState, type ReactNode } from 'react';

import Logo from '../Navigation/Logo';

export default function SideBar({
  collapsed: serverCollapsed,
  children,
}: {
  collapsed: boolean;
  children: ReactNode;
}) {
  const [collapsed, setCollapsed] = useState(serverCollapsed);
  const sideBarRef = useRef<HTMLDivElement>(null);

  const [transition, setTransition] = useState(false);

  useEffect(() => {
    //
    // Menus need to overflow, but it looks bad on transition for items to overflow
    if (sideBarRef.current) {
      const startTransition = () => setTransition(true);
      const endTransition = () => setTransition(false);

      sideBarRef.current.addEventListener('transitionstart', startTransition);
      sideBarRef.current.addEventListener('transitionend', endTransition);

      return () => {
        sideBarRef.current?.removeEventListener('transitionstart', startTransition);
        sideBarRef.current?.removeEventListener('transitionstart', endTransition);
      };
    }
  }, []);

  return (
    <div
      ref={sideBarRef}
      className={`bg-canvasBase border-subtle sticky top-0 z-[100] flex h-screen flex-col justify-start ${
        collapsed ? 'w-[64px]' : 'w-[224px]'
      }  shrink-0 border-r transition-[width] delay-150 duration-300 ${
        transition ? 'overflow-hidden' : 'overflow-visible'
      }`}
    >
      <Logo collapsed={collapsed} setCollapsed={setCollapsed} />
      {children}
    </div>
  );
}
