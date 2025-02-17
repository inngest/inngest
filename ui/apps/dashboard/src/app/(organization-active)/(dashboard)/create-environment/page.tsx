import { Header } from '@inngest/components/Header/Header';

import Layout from '@/components/Layout/Layout';
import CreateEnvironment from './CreateEnvironment';

export default async function Create() {
  return (
    <Layout>
      <div className="flex-col">
        <Header breadcrumb={[{ text: 'Environments', href: '/env' }, { text: 'Create' }]} />
        <div className="no-scrollbar overflow-y-scroll p-6">
          <CreateEnvironment />
        </div>
      </div>
    </Layout>
  );
}
