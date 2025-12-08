"use server";

import { Suspense, type ReactNode } from "react";

import IncidentBanner from "@/app/(organization-active)/IncidentBanner";
import { getNavCollapsed } from "@/app/actions";
// import { BillingBanner } from '@/components/BillingBanner';
// import { ExecutionOverageBanner } from '@/components/ExecutionOverage';
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
        <IncidentBanner />

        <Suspense>{/* <BillingBanner /> */}</Suspense>

        <Suspense>{/* <ExecutionOverageBanner /> */}</Suspense>

        {children}
      </div>
    </div>
  );
}
