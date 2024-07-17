'use client';

import { useState, type ReactNode } from 'react';

import Logo from '../Navigation/Logo';

export default function SideBar({
  collapsed: serverCollapsed,
  children,
}: {
  collapsed: boolean;
  children: ReactNode;
}) {
  const [collapsed, setCollapsed] = useState(serverCollapsed);
  return (
    <div
      className={`bg-canvasBase border-subtle h-screen ${
        collapsed ? 'w-[64px]' : 'w-[224px]'
      } shrink-0 border-r transition-all delay-150 duration-300`}
    >
      <Logo collapsed={collapsed} setCollapsed={setCollapsed} />
      {children}
    </div>
  );
}
