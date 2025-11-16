import { type ReactNode } from "react";

import { ProfileDisplayType } from "@/queries/server-only/profile";
import type { Environment } from "@/utils/environments";
import { Header } from "@inngest/components/Header/NewHeader";
import { useRouterState } from "@tanstack/react-router";
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
  const layoutHeader = useRouterState({
    //
    // get the last, fully merged context
    select: (s) => s.matches[s.matches.length - 1]?.context?.layoutHeader,
  });
  return (
    <div
      id="layout-scroll-container"
      className="fixed z-50 flex h-screen w-full flex-row justify-start overflow-y-scroll overscroll-y-none"
    >
      <SideBar activeEnv={activeEnv} collapsed={collapsed} profile={profile} />

      <div className="no-scrollbar flex w-full flex-col overflow-x-scroll">
        {/* TANSTACK TODO: add incident banner, billing banner, and execution overage banner here */}
        {layoutHeader && <Header {...layoutHeader} />}
        {children}
      </div>
    </div>
  );
}
