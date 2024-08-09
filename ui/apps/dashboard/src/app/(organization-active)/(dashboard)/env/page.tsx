import Environments from '@/components/Environments/Environments';
import OldEnvs from '@/components/Environments/old/oldPage';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { Header } from '@/components/Header/Header';
import Layout from '@/components/Layout/Layout';
import AppNavigation from '@/components/Navigation/old/AppNavigation';

export default async function EnvsPage() {
  const newIANav = await getBooleanFlag('new-ia-nav');

  return newIANav ? (
    <Layout>
      <div className="flex-col">
        <Header backNav={true} breadcrumb={[{ text: 'Environments' }]} />
        <div className="no-scrollbar overflow-y-scroll px-6">
          <Environments />
        </div>
      </div>
    </Layout>
  ) : (
    <>
      <AppNavigation envSlug="all" />
      <OldEnvs />
    </>
  );
}
