import Environments from '@/components/Environments/Environments';
import OldEnvs from '@/components/Environments/old/oldPage';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import Layout from '@/components/Layout/Layout';
import AppNavigation from '@/components/Navigation/old/AppNavigation';

export default async function EnvsPage() {
  const newIANav = await getBooleanFlag('new-ia-nav');

  return newIANav ? (
    <Layout>
      <Environments />
    </Layout>
  ) : (
    <>
      <AppNavigation envSlug="all" />
      <OldEnvs />
    </>
  );
}
