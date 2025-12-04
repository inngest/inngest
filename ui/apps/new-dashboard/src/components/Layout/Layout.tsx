import { type ReactNode } from "react";

import { type ProfileDisplayType } from "@/queries/server/profile";
import type { Environment } from "@/utils/environments";
import SideBar from "./SideBar";

type LayoutProps = {
  collapsed: boolean | undefined;
  activeEnv?: Environment;
  profile?: ProfileDisplayType;
  children: ReactNode;
};

export default function Layout({
  collapsed,
  activeEnv,
  profile,
  children,
}: LayoutProps) {
  return (
    <div
      id="layout-scroll-container"
      className="fixed z-50 flex h-screen w-full flex-row justify-start overflow-y-scroll overscroll-y-none"
    >
      <SideBar activeEnv={activeEnv} collapsed={collapsed} profile={profile} />

      <div className="no-scrollbar flex w-full flex-col overflow-x-scroll">
        {/* TANSTACK TODO: add incident banner, billing banner, and execution overage banner here */}
        {children}
      </div>
    </div>
  );
}
