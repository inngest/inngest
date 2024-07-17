import Environments from '@/components/Environments/Environments';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import Layout from '@/components/Layout/Layout';
import getAllEnvironments from '@/queries/server-only/getAllEnvironments';
import OldEnvs from '../../../../components/Environments/old/oldPage';

export default async function EnvsPage() {
  const envs = await getAllEnvironments();
  const newIANav = await getBooleanFlag('new-ia-nav');

  return true ? (
    <Layout>
      <Environments envs={envs} />
    </Layout>
  ) : (
    <OldEnvs environments={envs} />
  );
}
