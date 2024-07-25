'use server';

import { type ReactNode } from 'react';
import { cookies } from 'next/headers';

import IncidentBanner from '@/app/(organization-active)/IncidentBanner';
import type { Environment } from '@/utils/environments';
import { Help } from '../Navigation/Help';
import { Integrations } from '../Navigation/Integrations';
import Navigation from '../Navigation/Navigation';
import { Profile } from '../Navigation/Profile';
import SideBar from './SideBar';

type LayoutProps = {
  envSlug?: string;
  activeEnv?: Environment;
  children: ReactNode;
};

export default async function Layout({ activeEnv, children }: LayoutProps) {
  const cookieStore = cookies();
  const collapsed = cookieStore.get('navCollapsed')?.value === 'true';

  return (
    <div className="flex w-full flex-row justify-start">
      <SideBar collapsed={collapsed}>
        <div className="flex grow flex-col justify-between">
          <Navigation collapsed={collapsed} activeEnv={activeEnv} />

          <div>
            <Integrations collapsed={collapsed} />
            <Help collapsed={collapsed} />
            <Profile collapsed={collapsed} />
          </div>
        </div>
      </SideBar>

      <div className="flex w-full flex-col">
        <IncidentBanner />
        {children}
      </div>
    </div>
  );
}
