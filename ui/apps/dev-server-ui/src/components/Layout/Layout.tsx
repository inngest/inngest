'use client';

import React, { type ReactNode } from 'react';

import SideBar from './SideBar';

export default function Layout({ children }: { children: ReactNode }) {
  const collapsed = localStorage.getItem('navCollapsed') === 'true';

  return (
    <div className="flex w-full flex-row justify-start">
      <SideBar collapsed={collapsed} />

      <div className="flex w-full flex-col">{children}</div>
    </div>
  );
}
