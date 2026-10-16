import { Header } from '@inngest/components/Header/Header';
import { createFileRoute, Outlet } from '@tanstack/react-router';

import PageTitle from '@/components/Billing/PageTitle';
import { MarketplaceAccessControl } from '@/components/Billing/MarketplaceAccessControl';
import { getProfileDisplay } from '@/queries/server/profile';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute('/_authed/billing')({
  component: BillingLayout,
  loader: async () => {
    const profile = await getProfileDisplay();
    return { profile };
  },
});

function BillingLayout() {
  const { profile } = Route.useLoaderData();

  let tabs = [
    {
      children: 'Overview',
      href: pathCreator.billing(),
      exactRouteMatch: true,
    },
    {
      children: 'Usage',
      href: pathCreator.billing({ tab: 'usage' }),
    },
    {
      children: 'Payments',
      href: pathCreator.billing({ tab: 'payments' }),
    },
    {
      children: 'Plans',
      href: pathCreator.billing({ tab: 'plans' }),
    },
  ];

  if (profile.isMarketplace) {
    //
    // The usage page is the only billing page that marketplace accounts can
    // access, so no need for tabs
    tabs = [];
  }

  return (
    <>
      <Header
        backNav={true}
        breadcrumb={[{ text: 'Billing', href: pathCreator.billing() }]}
        tabs={tabs}
      />
      <div className="bg-canvasSubtle no-scrollbar min-h-0 flex-1 overflow-y-auto">
        <MarketplaceAccessControl isMarketplace={profile.isMarketplace}>
          <div className="mx-auto w-full max-w-[1200px] px-6 pb-16">
            <PageTitle />
            <Outlet />
          </div>
        </MarketplaceAccessControl>
      </div>
    </>
  );
}
