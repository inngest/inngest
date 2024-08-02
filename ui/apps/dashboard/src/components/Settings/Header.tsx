'use client';

import { usePathname } from 'next/navigation';

import { Header } from '@/components/Header/Header';

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

export const SettingsHeader = () => {
  const pathname = usePathname();
  const defined = <T,>(value: T | undefined) => value !== undefined;

  const breadcrumb = paths
    .map(([path, text]) =>
      pathname?.endsWith(path) ? { text, href: `/settings${path}` } : undefined
    )
    .filter(defined);

  return <Header backNav={true} breadcrumb={breadcrumb} />;
};
