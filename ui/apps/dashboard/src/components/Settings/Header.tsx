'use client';

import { usePathname } from 'next/navigation';
import { Header, type BreadCrumbType } from '@inngest/components/Header/Header';

//
// In the new IA, all the settings pages
// are their own top level pages with their own breadcrumb
const paths: [string, string][] = [
  ['/integrations', 'Integrations'],
  ['/billing', 'Billing'],
  ['/organization-settings', 'Organization'],
  ['/organization', 'Members'],
  ['/user', 'Your profile'],
];

const defined = <T,>(value: T | undefined): value is T => value !== undefined;

const getBreadCrumbs = (pathname: string): BreadCrumbType[] =>
  pathname.includes('integrations/vercel')
    ? [{ text: 'Integrations', href: `/settings/integrations` }, { text: 'Vercel' }]
    : pathname.includes('integrations/neon')
    ? [{ text: 'Integrations', href: `/settings/integrations` }, { text: 'Neon' }]
    : paths.map(([path, text]) => (pathname.endsWith(path) ? { text } : undefined)).filter(defined);

export const SettingsHeader = () => {
  const pathname = usePathname();
  const breadcrumb: BreadCrumbType[] = getBreadCrumbs(pathname);

  return <Header backNav={true} breadcrumb={breadcrumb} />;
};
