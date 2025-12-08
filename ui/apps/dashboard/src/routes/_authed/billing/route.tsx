import { Header } from '@inngest/components/Header/NewHeader';
import { createFileRoute, Outlet } from '@tanstack/react-router';

import PageTitle from '@/components/Billing/PageTitle';
import { MarketplaceAccessControl } from '@/components/Billing/MarketplaceAccessControl';
import { getProfileDisplay } from '@/queries/server/profile';
import { pathCreator } from '@/utils/urls';
import Toaster from '@/components/Toast/Toaster';

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
      <div className="bg-canvasSubtle flex h-full flex-col">
        <MarketplaceAccessControl isMarketplace={profile.isMarketplace}>
          <div className="no-scrollbar mx-auto w-full max-w-[1200px] overflow-y-scroll px-6 pb-16">
            <PageTitle />
            <Outlet />
          </div>
        </MarketplaceAccessControl>
        <Toaster />
      </div>
    </>
  );
}
