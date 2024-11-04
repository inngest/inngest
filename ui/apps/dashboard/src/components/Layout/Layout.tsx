'use server';

import { type ReactNode } from 'react';

import IncidentBanner from '@/app/(organization-active)/IncidentBanner';
import { getNavCollapsed } from '@/app/actions';
import { BillingBanner } from '@/components/BillingBanner';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getProfileDisplay } from '@/queries/server-only/profile';
import type { Environment } from '@/utils/environments';
import SideBar from './SideBar';

type LayoutProps = {
  envSlug?: string;
  activeEnv?: Environment;
  children: ReactNode;
};

export default async function Layout({ activeEnv, children }: LayoutProps) {
  const collapsed = await getNavCollapsed();
  const profile = await getProfileDisplay();

  let entitlementUsage;
  try {
    entitlementUsage = (await graphqlAPI.request(entitlementUsageQuery)).account.entitlementUsage;
  } catch (e) {
    console.error(e);
  }

  return (
    <div className="fixed z-50 flex h-screen w-full flex-row justify-start overflow-y-scroll overscroll-y-none">
      <SideBar collapsed={collapsed} activeEnv={activeEnv} profile={profile} />

      <div className="flex w-full flex-col overflow-x-scroll">
        <IncidentBanner />
        {entitlementUsage && <BillingBanner entitlementUsage={entitlementUsage} />}

        {children}
      </div>
    </div>
  );
}

const entitlementUsageQuery = graphql(`
  query EntitlementUsage {
    account {
      id
      entitlementUsage {
        runCount {
          current
          limit
        }
      }
    }
  }
`);
