'use client';

import { type ReactNode } from 'react';

import SideBar from './SideBar';

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="no-scrollbar fixed z-50 flex h-screen w-full flex-row justify-start overflow-y-scroll overscroll-y-none">
      <SideBar />

      <div className="flex w-full flex-col">{children}</div>
    </div>
  );
}
