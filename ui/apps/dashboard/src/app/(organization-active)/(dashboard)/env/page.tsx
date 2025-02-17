import { Header } from '@inngest/components/Header/Header';

import Environments from '@/components/Environments/Environments';
import Layout from '@/components/Layout/Layout';

export default async function EnvsPage() {
  return (
    <Layout>
      <div className="flex-col">
        <Header backNav={true} breadcrumb={[{ text: 'Environments' }]} />
        <div className="no-scrollbar overflow-y-scroll px-6">
          <Environments />
        </div>
      </div>
    </Layout>
  );
}
