'use client';

import { useState, type ReactNode } from 'react';
import { HelpIcon } from '@inngest/components/icons/sections/Help';
import { IntegrationsIcon } from '@inngest/components/icons/sections/Integrations';

import Logo from '../Navigation/Logo';
import { MenuItem } from '../Navigation/MenuItem';
import { Profile } from '../Navigation/Profile';

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
      className={`bg-canvasBase border-subtle flex h-screen flex-col justify-start ${
        collapsed ? 'w-[64px]' : 'w-[224px]'
      } shrink-0 border-r transition-all delay-150 duration-300`}
    >
      <Logo collapsed={collapsed} setCollapsed={setCollapsed} />
      {children}
    </div>
  );
}
