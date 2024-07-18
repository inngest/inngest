'use server';

import { Suspense, type ReactNode } from 'react';
import { cookies } from 'next/headers';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';

import IncidentBanner from '@/app/(organization-active)/IncidentBanner';
import type { Environment } from '@/utils/environments';
import Navigation from '../Navigation/Navigation';
import SideBar from './SideBar';

type LayoutProps = {
  envSlug?: string;
  envs?: Environment[];
  activeEnv?: Environment;
  children: ReactNode;
};

export default async function Layout({ envs, activeEnv, children }: LayoutProps) {
  const cookieStore = cookies();
  const collapsed = cookieStore.get('navCollapsed')?.value === 'true';

  return (
    <div className="flex w-full flex-row justify-start">
      <SideBar collapsed={collapsed}>
        <Suspense fallback={<Skeleton className="h-full w-[12rem]" />}>
          <Navigation collapsed={collapsed} envs={envs} activeEnv={activeEnv} />
        </Suspense>
      </SideBar>
      <div className="flex w-full flex-col">
        <IncidentBanner />
        {children}
      </div>
    </div>
  );
}
