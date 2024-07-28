'use server';

import { type ReactNode } from 'react';

import IncidentBanner from '@/app/(organization-active)/IncidentBanner';
import { getNavCollapsed } from '@/app/actions';
import { getProfile } from '@/queries/server-only/profile';
import type { Environment } from '@/utils/environments';
import SideBar from './SideBar';

type LayoutProps = {
  envSlug?: string;
  activeEnv?: Environment;
  children: ReactNode;
};

export default async function Layout({ activeEnv, children }: LayoutProps) {
  const collapsed = await getNavCollapsed();
  const { user, org } = await getProfile();
  const profile = { orgName: org?.name, fullName: `${user.firstName} ${user.lastName}` };

  return (
    <div className="flex w-full flex-row justify-start">
      <SideBar collapsed={collapsed} activeEnv={activeEnv} profile={profile} />

      <div className="flex w-full flex-col">
        <IncidentBanner />
        {children}
      </div>
    </div>
  );
}
