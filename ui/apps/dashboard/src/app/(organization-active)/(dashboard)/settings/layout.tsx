import { RiSettings3Line } from '@remixicon/react';

import Header from '@/components/Header/Header';
import AppNavigation from '@/components/Navigation/AppNavigation';
import Toaster from '@/components/Toaster';
import getAllEnvironments from '@/queries/server-only/getAllEnvironments';

type SettingsLayoutProps = {
  children: React.ReactNode;
};

export default async function SettingsLayout({ children }: SettingsLayoutProps) {
  const envs = await getAllEnvironments();

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

  return (
    <div className="flex h-full flex-col">
      <AppNavigation envs={envs} />
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
