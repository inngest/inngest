import { Cog6ToothIcon } from '@heroicons/react/20/solid';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import Header from '@/components/Header/Header';
import AppNavigation from '@/components/Navigation/AppNavigation';
import Toaster from '@/components/Toaster';

type SettingsLayoutProps = {
  children: React.ReactNode;
};

const DEFAULT_ENV_SLUG = 'production';

export default async function SettingsLayout({ children }: SettingsLayoutProps) {
  const isOrganizationsEnabled = await getBooleanFlag('organizations');

  const navLinks = [
    {
      href: '/settings/account',
      text: 'Account',
    },
    {
      href: '/settings/billing',
      text: 'Billing',
    },
    {
      href: '/settings/integrations',
      text: 'Integrations',
    },
    {
      href: '/settings/team',
      text: 'Team Management',
    },
  ];

  if (isOrganizationsEnabled) {
    navLinks.splice(1, 0, {
      href: '/settings/organization',
      text: 'Organization',
    });
    navLinks.pop();
  }

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
