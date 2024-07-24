import { EnvLayout } from '@/components/Environments/EnvLayout';
import Environments from '@/components/Environments/Environments';
import OldEnvs from '@/components/Environments/old/oldPage';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import AppNavigation from '@/components/Navigation/old/AppNavigation';

export default async function EnvsPage() {
  const newIANav = await getBooleanFlag('new-ia-nav');

  return newIANav ? (
    <EnvLayout>
      <Environments />
    </EnvLayout>
  ) : (
    <>
      <AppNavigation envSlug="all" />
      <OldEnvs />
    </>
  );
}
