import { Header } from '@inngest/components/Header/Header';

import Layout from '@/components/Layout/Layout';
import Toaster from '@/components/Toaster';

export default async function BillingLayout({ children }: React.PropsWithChildren) {
  return (
    <Layout>
      <div className="flex-col">
        <Header
          breadcrumb={[{ text: 'Billing', href: '/billing' }]}
          tabs={[
            {
              children: 'Usage',
              href: `/billing/usage`,
              exactRouteMatch: true,
            },
          ]}
        />
        <div className="no-scrollbar overflow-y-scroll px-6">{children}</div>
        <Toaster />
      </div>
    </Layout>
  );
}
