import Environments from '@/components/Environments/Environments';
import OldEnvs from '@/components/Environments/old/oldPage';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import Layout from '@/components/Layout/Layout';
import AppNavigation from '@/components/Navigation/old/AppNavigation';

export default async function EnvsPage() {
  const newIANav = await getBooleanFlag('new-ia-nav');

  return newIANav ? (
    <Layout>
      <div className="border-subtle flex h-[52px] w-full shrink-0 flex-row items-center justify-start border-b px-6">
        <div className="text-basis text-base leading-tight">All Environments</div>
      </div>

      <Environments />
    </Layout>
  ) : (
    <>
      <AppNavigation envSlug="all" />
      <OldEnvs />
    </>
  );
}
