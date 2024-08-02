'use client';

import { usePathname } from 'next/navigation';

import { Header, type BreadCrumbType } from '@/components/Header/Header';

//
// In the new IA, all the settings pages
// are their own top level pages with their own breadcrumb
const paths: [string, string][] = [
  ['/integrations', 'Integrations'],
  ['/billing', 'Billing'],
  ['/organization-settings', 'Oragnization'],
  ['/organization', 'Members'],
  ['/user', 'Your profile'],
];

const defined = <T,>(value: T | undefined): value is T => value !== undefined;

const getBreadCrumbs = (pathname: string): BreadCrumbType[] =>
  paths
    .map(([path, text]) =>
      pathname.endsWith(path) ? { text, href: `/settings${path}` } : undefined
    )
    .filter(defined);

export const SettingsHeader = () => {
  const pathname = usePathname();
  const breadcrumb: BreadCrumbType[] = getBreadCrumbs(pathname);

  return <Header backNav={true} breadcrumb={breadcrumb} />;
};
