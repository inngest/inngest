'use server';

import { Suspense, type ReactNode } from 'react';
import { cookies } from 'next/headers';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { HelpIcon } from '@inngest/components/icons/sections/Help';
import { IntegrationsIcon } from '@inngest/components/icons/sections/Integrations';

import IncidentBanner from '@/app/(organization-active)/IncidentBanner';
import type { Environment } from '@/utils/environments';
import { MenuItem } from '../Navigation/MenuItem';
import Navigation from '../Navigation/Navigation';
import { Profile } from '../Navigation/Profile';
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
          <div className="flex h-screen flex-col justify-between">
            <Navigation collapsed={collapsed} envs={envs} activeEnv={activeEnv} />
            <div>
              <div className="m-2.5">
                <MenuItem
                  href="/settings/integrations"
                  collapsed={collapsed}
                  text="Integrations"
                  icon={<IntegrationsIcon className="w-5" />}
                />
                <MenuItem
                  href="/support"
                  collapsed={collapsed}
                  text="Help and Feedback"
                  icon={<HelpIcon className="w-5" />}
                />
              </div>
              <Profile collapsed={collapsed} />
            </div>
          </div>
        </Suspense>
      </SideBar>
      <div className="flex w-full flex-col">
        <IncidentBanner />
        {children}
      </div>
    </div>
  );
}
