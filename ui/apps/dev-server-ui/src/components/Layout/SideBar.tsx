'use client';

import React, { useState } from 'react';

import { EnvironmentMenu } from '../Navigation/Environments';
import Manage from '../Navigation/Manage';
import Monitor from '../Navigation/Monitor';
import Logo from './Logo';

export default function SideBar({ collapsed: initCollapsed }: { collapsed: boolean }) {
  const [collapsed, setCollapsed] = useState(initCollapsed);

  return (
    <nav
      className={`bg-canvasBase border-subtle group
         top-0 flex h-screen flex-col justify-start ${
           collapsed ? 'w-[64px]' : 'w-[224px]'
         }  sticky z-[49] shrink-0 overflow-visible border-r`}
    >
      <Logo collapsed={collapsed} setCollapsed={setCollapsed} />
      <div className="flex grow flex-col justify-start pl-4 pr-4 pt-5">
        <EnvironmentMenu collapsed={collapsed} />
        <Monitor collapsed={collapsed} />
        <Manage collapsed={collapsed} />
        <div>help</div>
      </div>
    </nav>
  );
}
