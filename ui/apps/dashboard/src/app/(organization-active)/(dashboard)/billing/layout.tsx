import { Header } from '@inngest/components/Header/Header';

import PageTitle from '@/components/Billing/PageTitle';
import Layout from '@/components/Layout/Layout';
import Toaster from '@/components/Toaster';
import { getProfileDisplay } from '@/queries/server-only/profile';
import { pathCreator } from '@/utils/urls';
import MarketplaceAccessControl from './MarketplaceAccessControl';

export default async function BillingLayout({ children }: React.PropsWithChildren) {
  const profile = await getProfileDisplay();

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
    // The usage page is the only billing page that marketplace accounts can
    // access, so no need for tabs
    tabs = [];
  }

  return (
    <Layout>
      <div className="bg-canvasSubtle flex h-full flex-col">
        <Header
          backNav={true}
          breadcrumb={[{ text: 'Billing', href: pathCreator.billing() }]}
          tabs={tabs}
        />
        <MarketplaceAccessControl isMarketplace={profile.isMarketplace}>
          <div className="no-scrollbar mx-auto w-full max-w-[1200px] overflow-y-scroll px-6 pb-16">
            <PageTitle />
            {children}
          </div>
        </MarketplaceAccessControl>
        <Toaster />
      </div>
    </Layout>
  );
}
