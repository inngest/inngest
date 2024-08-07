'use client';

import React, { useEffect, useState } from 'react';

import { EnvironmentMenu } from '../Navigation/Environments';
import Manage from '../Navigation/Manage';
import Monitor from '../Navigation/Monitor';
import Logo from './Logo';

export default function SideBar() {
  const [collapsed, setCollapsed] = useState(true);

  useEffect(() => {
    //
    // TODO: something better so we don't flash
    typeof window !== 'undefined' &&
      window.localStorage.getItem('navCollapsed') === 'false' &&
      setCollapsed(false);
  }, []);

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
