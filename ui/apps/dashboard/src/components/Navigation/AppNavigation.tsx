import { type Route } from 'next';
import Link from 'next/link';
import {
  CodeBracketSquareIcon,
  Cog6ToothIcon,
  RocketLaunchIcon,
  WrenchIcon,
} from '@heroicons/react/20/solid';

import InngestLogo from '@/icons/InngestLogo';
import EventIcon from '@/icons/event.svg';
import AccountDropdown from './AccountDropdown';
import EnvironmentSelectMenu from './EnvironmentSelectMenu';
import NavItem from './NavItem';
import Navigation from './Navigation';
import SearchNavigation from './SearchNavigation';

type AppNavigationProps = {
  environmentSlug: string;
};
type NavItem = {
  href: string;
  text: string;
  icon?: React.ReactNode;
  hide: string[];
};

const ALL_ENVIRONMENTS_SLUG = 'all';
const BRANCH_PARENT_SLUG = 'branch';

export default function AppNavigation({ environmentSlug }: AppNavigationProps) {
  const items: NavItem[] = [
    {
      href: `/env/${environmentSlug}/functions`,
      text: 'Functions',
      hide: [ALL_ENVIRONMENTS_SLUG, BRANCH_PARENT_SLUG],
      icon: <CodeBracketSquareIcon className="w-4" />,
    },
    {
      href: `/env/${environmentSlug}/events`,
      text: 'Events',
      hide: [ALL_ENVIRONMENTS_SLUG, BRANCH_PARENT_SLUG],
      icon: <EventIcon className="w-5" />,
    },
    {
      href: `/env/${environmentSlug}/deploys`,
      text: 'Deploys',
      hide: [ALL_ENVIRONMENTS_SLUG],
      icon: <RocketLaunchIcon className="w-3.5" />,
    },
    {
      href: `/env/${environmentSlug}/manage`,
      text: 'Manage',
      hide: [ALL_ENVIRONMENTS_SLUG],
      icon: <WrenchIcon className="w-3.5" />,
    },
  ];
  const visibleItems = items.filter((item) => !item.hide.includes(environmentSlug));

  return (
    <nav className="bg-slate-940 left-0 right-0 top-0 z-50 flex w-full items-center justify-between pl-6">
      <div className="flex h-12 items-center gap-3">
        <Link href={process.env.NEXT_PUBLIC_HOME_PATH as Route}>
          <InngestLogo className="mr-2 mt-0.5 text-white" width={66} />
        </Link>
        <EnvironmentSelectMenu environmentSlug={environmentSlug} />
        <Navigation>
          {visibleItems.map(({ href, text, icon }) => (
            <NavItem key={href} href={href as Route} icon={icon} text={text} />
          ))}
        </Navigation>
      </div>
      <SearchNavigation />
      <AccountDropdown />
    </nav>
  );
}
