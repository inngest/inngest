import { type Route } from 'next';
import Link from 'next/link';
import {
  CodeBracketSquareIcon,
  MagnifyingGlassIcon,
  RocketLaunchIcon,
  Squares2X2Icon,
  WrenchIcon,
} from '@heroicons/react/20/solid';
import { Badge } from '@inngest/components/Badge';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import OrganizationDropdown from '@/components/Navigation/OrganizationDropdown';
import { graphql } from '@/gql';
import InngestLogo from '@/icons/InngestLogo';
import EventIcon from '@/icons/event.svg';
import graphqlAPI from '@/queries/graphqlAPI';
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
  badge?: React.ReactNode;
  hide: string[];
};

const ALL_ENVIRONMENTS_SLUG = 'all';
const BRANCH_PARENT_SLUG = 'branch';

const GetAccountCreationTimeDocument = graphql(`
  query GetAccountCreationTime {
    account {
      createdAt
    }
  }
`);

// TODO: Delete this when the deploys page is fully deleted
async function isDeploysVisible() {
  const { account } = await graphqlAPI.request(GetAccountCreationTimeDocument);

  const appsPageLaunchDate = new Date('2024-01-23T15:00:00.000Z');
  return new Date(account.createdAt) < appsPageLaunchDate;
}

export default async function AppNavigation({ environmentSlug }: AppNavigationProps) {
  const isEventSearchEnabled = await getBooleanFlag('event-search');
  const isOrganizationsEnabled = await getBooleanFlag('organizations');

  let items: NavItem[] = [
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
      href: `/env/${environmentSlug}/manage`,
      text: 'Manage',
      hide: [ALL_ENVIRONMENTS_SLUG],
      icon: <WrenchIcon className="w-3.5" />,
    },
  ];

  if (await isDeploysVisible()) {
    // Insert the "Deploys" item after the 2nd item.
    items = [
      ...items.slice(0, 2),
      {
        href: `/env/${environmentSlug}/deploys`,
        text: 'Deploys',
        hide: [ALL_ENVIRONMENTS_SLUG],
        icon: <RocketLaunchIcon className="w-3.5" />,
      },
      ...items.slice(2),
    ];
  }

  if (await getBooleanFlag('apps-page')) {
    // Insert the "Apps" item after the 1st item.
    items = [
      {
        href: `/env/${environmentSlug}/apps`,
        text: 'Apps',
        hide: [ALL_ENVIRONMENTS_SLUG],
        icon: <Squares2X2Icon className="w-3.5" />,
        badge: (
          <Badge kind="solid" className=" h-3.5 bg-indigo-500 px-[0.235rem] text-white">
            New
          </Badge>
        ),
      },
      ...items,
    ];
  }

  if (isEventSearchEnabled) {
    // Insert the "Event Search" item after the 3rd item.
    items = [
      ...items.slice(0, 3),
      {
        href: `/env/${environmentSlug}/event-search`,
        text: 'Event Search',
        hide: [ALL_ENVIRONMENTS_SLUG, BRANCH_PARENT_SLUG],
        icon: <MagnifyingGlassIcon className="w-3.5" />,
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
        <EnvironmentSelectMenu environmentSlug={environmentSlug} />
        <Navigation>
          {visibleItems.map(({ href, text, icon, badge }) => (
            <NavItem key={href} href={href as Route} icon={icon} text={text} badge={badge} />
          ))}
        </Navigation>
      </div>
      <div className="flex h-full items-center">
        <SearchNavigation />
        {isOrganizationsEnabled && <OrganizationDropdown />}
        <AccountDropdown />
      </div>
    </nav>
  );
}
