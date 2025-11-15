import { type ReactNode } from "react";

import { getNavCollapsed } from "@/app/actions";

import { getProfileDisplay } from "@/queries/server-only/profile";
import type { Environment } from "@/utils/environments";
import SideBar from "./SideBar";

type LayoutProps = {
  activeEnv?: Environment;
  children: ReactNode;
};

export default async function Layout({ activeEnv, children }: LayoutProps) {
  const collapsed = await getNavCollapsed();
  const profile = await getProfileDisplay();

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
