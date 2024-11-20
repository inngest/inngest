import { Header } from '@inngest/components/Header/Header';

import PageTitle from '@/components/Billing/PageTitle';
import Layout from '@/components/Layout/Layout';
import Toaster from '@/components/Toaster';
import { pathCreator } from '@/utils/urls';

export default async function BillingLayout({ children }: React.PropsWithChildren) {
  return (
    <Layout>
      <div className="bg-canvasSubtle flex h-full flex-col">
        <Header
          backNav={true}
          breadcrumb={[{ text: 'Billing', href: pathCreator.billing() }]}
          tabs={[
            {
              children: 'Overview',
              href: pathCreator.billing(),
              exactRouteMatch: true,
            },
            {
              children: 'Usage',
              href: pathCreator.billingUsage(),
            },
            {
              children: 'Payments',
              href: pathCreator.billingPayments(),
            },
            {
              children: 'Plans',
              href: pathCreator.billingPlans(),
            },
          ]}
        />
        <div className="no-scrollbar mx-auto w-full max-w-[1200px] overflow-y-scroll px-6 pb-16">
          <PageTitle />
          {children}
        </div>
        <Toaster />
      </div>
    </Layout>
  );
}
