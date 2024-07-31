import { RiSettings3Line } from '@remixicon/react';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import Header from '@/components/Header/Header';
import Layout from '@/components/Layout/Layout';
import AppNavigation from '@/components/Navigation/old/AppNavigation';
import Toaster from '@/components/Toaster';

type SettingsLayoutProps = {
  children: React.ReactNode;
};

export default async function SettingsLayout({ children }: SettingsLayoutProps) {
  const newIANav = await getBooleanFlag('new-ia-nav');
  const navLinks = [
    {
      href: '/settings/user',
      text: 'User',
    },
    {
      href: '/settings/organization',
      text: 'Organization',
    },
    {
      href: '/settings/billing',
      text: 'Billing',
    },
    {
      href: '/settings/integrations',
      text: 'Integrations',
    },
  ];

  return newIANav ? (
    <Layout>
      <Header
        title="Settings"
        links={navLinks}
        icon={<RiSettings3Line className="w-4 text-white" />}
      />
      <div className="px-6">{children}</div>
      <Toaster />
    </Layout>
  ) : (
    <div className="flex h-full flex-col">
      <AppNavigation envSlug="all" />
      <Header
        title="Settings"
        links={navLinks}
        icon={<RiSettings3Line className="w-4 text-white" />}
      />
      {children}
      <Toaster />
    </div>
  );
}
