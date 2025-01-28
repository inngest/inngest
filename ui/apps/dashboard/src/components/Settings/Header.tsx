'use client';

import { usePathname } from 'next/navigation';
import { Header, type BreadCrumbType } from '@inngest/components/Header/Header';

//
// In the new IA, all the settings pages
// are their own top level pages with their own breadcrumb
const paths: [string, string][] = [
  ['/integrations', 'Integrations'],
  ['/organization', 'Organization'],
  ['/organization/organization-members', 'Members'],
  ['/user', 'Profile'],
  ['/user/security', 'Profile'],
];

const defined = <T,>(value: T | undefined): value is T => value !== undefined;

const getBreadCrumbs = (pathname: string): BreadCrumbType[] =>
  pathname.includes('integrations/vercel')
    ? [{ text: 'Integrations', href: `/settings/integrations` }, { text: 'Vercel' }]
    : pathname.includes('integrations/neon')
    ? [{ text: 'Integrations', href: `/settings/integrations` }, { text: 'Neon' }]
    : paths.map(([path, text]) => (pathname.endsWith(path) ? { text } : undefined)).filter(defined);

const userTabs = [
  {
    children: 'General',
    href: '/settings/user',
    exactRouteMatch: true,
  },
  {
    children: 'Security',
    href: '/settings/user/security',
  },
];

export const SettingsHeader = () => {
  const pathname = usePathname();
  const breadcrumb: BreadCrumbType[] = getBreadCrumbs(pathname);
  const isProfilePage = pathname.includes('settings/user');

  return (
    <Header backNav={true} breadcrumb={breadcrumb} tabs={isProfilePage ? userTabs : undefined} />
  );
};
