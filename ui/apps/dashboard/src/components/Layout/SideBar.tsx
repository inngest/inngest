'use client';

import { useState } from 'react';

import type { Environment } from '@/utils/environments';
import { Help } from '../Navigation/Help';
import { Integrations } from '../Navigation/Integrations';
import Logo from '../Navigation/Logo';
import Navigation from '../Navigation/Navigation';
import { Profile } from '../Navigation/Profile';

export default function SideBar({
  collapsed: serverCollapsed,
  activeEnv,
  profile,
}: {
  collapsed: boolean;
  activeEnv?: Environment;
  profile: { orgName?: string; fullName: string };
}) {
  const [collapsed, setCollapsed] = useState(serverCollapsed);

  return (
    <nav
      className={`bg-canvasBase  border-subtle group
         top-0 flex h-screen flex-col justify-start ${
           collapsed ? 'w-[64px]' : 'w-[224px]'
         }  sticky z-[500] shrink-0 overflow-visible border-r`}
    >
      <Logo collapsed={collapsed} setCollapsed={setCollapsed} />
      <div className="flex grow flex-col justify-between">
        <Navigation collapsed={collapsed} activeEnv={activeEnv} />

        <div>
          <Integrations collapsed={collapsed} />
          <Help collapsed={collapsed} />
          <Profile collapsed={collapsed} profile={profile} />
        </div>
      </div>
    </nav>
  );
}
