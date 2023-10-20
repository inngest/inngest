import {
  Cog6ToothIcon,
  CreditCardIcon,
  CubeIcon,
  UserCircleIcon,
  UserGroupIcon,
} from '@heroicons/react/20/solid';

import Header from '@/components/Header/Header';
import AppNavigation from '@/components/Navigation/AppNavigation';
import Toaster from '@/components/Toaster';

type SettingsLayoutProps = {
  children: React.ReactNode;
};

const DEFAULT_ENV_SLUG = 'production';

export default function SettingsLayout({ children }: SettingsLayoutProps) {
  const navLinks = [
    {
      href: '/settings/account',
      text: 'Account',
      icon: <UserCircleIcon className="h-4" />,
    },
    {
      href: '/settings/billing',
      text: 'Billing',
      icon: <CreditCardIcon className="h-4" />,
    },
    {
      href: '/settings/integrations',
      text: 'Integrations',
      icon: <CubeIcon className="h-4" />,
    },
    {
      href: '/settings/team',
      text: 'Team Management',
      icon: <UserGroupIcon className="h-4" />,
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
