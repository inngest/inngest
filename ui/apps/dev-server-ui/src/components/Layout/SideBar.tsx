'use client';

import React from 'react';

import { EnvironmentMenu } from '../Navigation/Environments';
import { Help } from '../Navigation/Help';
import Manage from '../Navigation/Manage';
import Monitor from '../Navigation/Monitor';
import Logo from './Logo';

type SideBarProps = {
  collapsed: boolean;
  setCollapsed: (arg: boolean) => void;
};

export default function SideBar({ collapsed, setCollapsed }: SideBarProps) {
  return (
    <nav
      className={`bg-canvasBase border-subtle group
         top-0 flex h-screen flex-col justify-start ${
           collapsed ? 'w-[64px]' : 'w-[224px]'
         }  sticky z-[49] shrink-0 overflow-visible border-r`}
    >
      <Logo collapsed={collapsed} setCollapsed={setCollapsed} />
      <div className="flex grow flex-col justify-between">
        <div className="mx-4">
          <EnvironmentMenu collapsed={collapsed} />
          <Monitor collapsed={collapsed} />
          <Manage collapsed={collapsed} />
        </div>

        <div className="mb mx-4 mb-2">
          <Help collapsed={collapsed} />
        </div>
      </div>
    </nav>
  );
}
