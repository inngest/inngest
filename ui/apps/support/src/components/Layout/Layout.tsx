import { type ReactNode } from "react";

import SideBar from "./SideBar";
import { ProfileDisplayType } from "@/data/profile";

type LayoutProps = {
  collapsed: boolean | undefined;
  profile?: ProfileDisplayType;
  children: ReactNode;
};

export default function Layout({ collapsed, profile, children }: LayoutProps) {
  return (
    <div
      id="layout-scroll-container"
      className="fixed z-50 flex h-screen w-full flex-row justify-start overflow-y-scroll overscroll-y-none"
    >
      <SideBar collapsed={collapsed} profile={profile} />

      <div className="no-scrollbar flex w-full flex-col overflow-x-scroll">
        {children}
      </div>
    </div>
  );
}
