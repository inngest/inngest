'use client';

import { usePathname } from 'next/navigation';

import { pathCreator } from '@/utils/urls';

export default function PageTitle() {
  const pathname = usePathname();

  const routeTitles: { [key: string]: string } = {
    [pathCreator.billing()]: 'Overview',
    [pathCreator.billingUsage()]: 'Usage',
    [pathCreator.billingPayments()]: 'Payments',
    [pathCreator.billingPlans()]: 'Plans',
  };
  const pageTitle = routeTitles[pathname] || '';

  return <h2 className="my-9 text-2xl">{pageTitle}</h2>;
}
