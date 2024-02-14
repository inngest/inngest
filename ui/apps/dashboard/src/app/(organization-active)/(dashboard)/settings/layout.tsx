import { Cog6ToothIcon } from '@heroicons/react/20/solid';

import Header from '@/components/Header/Header';
import AppNavigation from '@/components/Navigation/AppNavigation';
import Toaster from '@/components/Toaster';

type SettingsLayoutProps = {
  children: React.ReactNode;
};

const DEFAULT_ENV_SLUG = 'production';

export default async function SettingsLayout({ children }: SettingsLayoutProps) {
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
      <AppNavigation environmentSlug={DEFAULT_ENV_SLUG} />
      <Header
        title="Settings"
        links={navLinks}
        icon={<Cog6ToothIcon className="w-4 text-white" />}
      />
      {children}
      <Toaster />
    </div>
  );
}
