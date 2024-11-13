import { Header } from '@inngest/components/Header/Header';

import Layout from '@/components/Layout/Layout';
import Toaster from '@/components/Toaster';

export default async function BillingLayout({ children }: React.PropsWithChildren) {
  return (
    <Layout>
      <div className="bg-canvasSubtle flex h-full flex-col">
        <Header
          backNav={true}
          breadcrumb={[{ text: 'Billing', href: '/billing' }]}
          tabs={[
            {
              children: 'Overview',
              href: `/billing`,
              exactRouteMatch: true,
            },
            {
              children: 'Usage',
              href: `/billing/usage`,
            },
          ]}
        />
        <div className="no-scrollbar mx-auto w-full max-w-[1200px] overflow-y-scroll px-6 pt-16">
          {children}
        </div>
        <Toaster />
      </div>
    </Layout>
  );
}
