import { redirect } from 'next/navigation';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import IntegrationsList from './integrations';

export default async function IntegrationsPage() {
  const newIntegrations = await getBooleanFlag('new-integrations');

  //
  // TODO: this can go away once the "new-integrations"
  // feature is fully live
  if (!newIntegrations) {
    redirect('/settings/integrations/vercel');
  }

  return <IntegrationsList />;
}
