'use client';

import { usePathname } from 'next/navigation';
import { NewLink } from '@inngest/components/Link/Link';

import { WEBSITE_PRICING_URL, pathCreator } from '@/utils/urls';

export default function PageTitle() {
  const pathname = usePathname();

  const routeTitles: { [key: string]: string } = {
    [pathCreator.billing()]: 'Overview',
    [pathCreator.billing({ tab: 'usage' })]: 'Usage',
    [pathCreator.billing({ tab: 'payments' })]: 'Payments',
    [pathCreator.billing({ tab: 'plans' })]: 'Plans',
  };
  const pageTitle = routeTitles[pathname] || '';
  const cta =
    pathname === pathCreator.billing({ tab: 'plans' }) ? (
      <NewLink
        target="_blank"
        size="small"
        href={WEBSITE_PRICING_URL + '?ref=app-billing-page-plans'}
      >
        View pricing page
      </NewLink>
    ) : null;

  return (
    <div className="flex items-center justify-between">
      <h2 className="my-9 text-2xl">{pageTitle}</h2>
      {cta}
    </div>
  );
}
