import { type Route } from 'next';
import Link from 'next/link';
import { IconApp } from '@inngest/components/icons/App';
import { IconEvent } from '@inngest/components/icons/Event';
import { IconFunction } from '@inngest/components/icons/Function';
import { RiSearchLine, RiToolsLine } from '@remixicon/react';

import OrganizationDropdown from '@/components/Navigation/old/OrganizationDropdown';
import UserDropdown from '@/components/Navigation/old/UserDropdown';
import InngestLogo from '@/icons/InngestLogo';
import type { Environment } from '@/utils/environments';
import { getBooleanFlag } from '../../FeatureFlags/ServerFeatureFlag';
import EnvironmentSelectMenu from './EnvironmentSelectMenu';
import NavItem from './NavItem';
import Navigation from './Navigation';
import SearchNavigation from './SearchNavigation';

type AppNavigationProps = {
  envs: Environment[];
  activeEnv?: Environment;
  slug?: string;
};
type NavItem = {
  href: string;
  text: string;
  icon?: React.ReactNode;
  badge?: React.ReactNode;
  hide: string[];
};

const ALL_ENVIRONMENTS_SLUG = 'all';
const BRANCH_PARENT_SLUG = 'branch';
const DEFAULT_ENV_SLUG = 'production';

export default async function AppNavigation({ envs, activeEnv, slug }: AppNavigationProps) {
  const isEventSearchEnabled = await getBooleanFlag('event-search');
  const environmentSlug = slug || DEFAULT_ENV_SLUG;

  let items: NavItem[] = [
    {
      href: `/env/${environmentSlug}/apps`,
      text: 'Apps',
      hide: [ALL_ENVIRONMENTS_SLUG],
      icon: <IconApp className="w-3.5" />,
    },
    {
      href: `/env/${environmentSlug}/functions`,
      text: 'Functions',
      hide: [ALL_ENVIRONMENTS_SLUG, BRANCH_PARENT_SLUG],
      icon: <IconFunction className="w-4" />,
    },
    {
      href: `/env/${environmentSlug}/events`,
      text: 'Events',
      hide: [ALL_ENVIRONMENTS_SLUG, BRANCH_PARENT_SLUG],
      icon: <IconEvent className="w-5" />,
    },
    {
      href: `/env/${environmentSlug}/manage`,
      text: 'Manage',
      hide: [ALL_ENVIRONMENTS_SLUG],
      icon: <RiToolsLine className="w-3.5" />,
    },
  ];

  if (isEventSearchEnabled) {
    // Insert the "Event Search" item after the 3rd item.
    items = [
      ...items.slice(0, 3),
      {
        href: `/env/${environmentSlug}/event-search`,
        text: 'Event Search',
        hide: [ALL_ENVIRONMENTS_SLUG, BRANCH_PARENT_SLUG],
        icon: <RiSearchLine className="w-3.5" />,
      },
      ...items.slice(3),
    ];
  }

  const visibleItems = items.filter((item) => !item.hide.includes(environmentSlug));

  return (
    <nav className="bg-slate-940 left-0 right-0 top-0 z-50 flex w-full items-center justify-between pl-6">
      <div className="flex h-12 items-center gap-3">
        <Link href={process.env.NEXT_PUBLIC_HOME_PATH as Route}>
          <InngestLogo className="mr-2 mt-0.5 text-white" width={66} />
        </Link>
        <EnvironmentSelectMenu envs={envs} activeEnv={activeEnv} />
        <Navigation>
          {visibleItems.map(({ href, text, icon, badge }) => (
            <NavItem key={href} href={href as Route} icon={icon} text={text} badge={badge} />
          ))}
        </Navigation>
      </div>
      <div className="flex h-full items-center">
        <SearchNavigation />
        <OrganizationDropdown />
        <UserDropdown />
      </div>
    </nav>
  );
}
