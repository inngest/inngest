import { PostgresIntegrations } from '@/components/PostgresIntegration/data';
import { getVercelIntegration } from './data';
import IntegrationsList from './integrations';

export default async function IntegrationsPage() {
  let allIntegrations: React.ComponentProps<typeof IntegrationsList>['integrations'] =
    await PostgresIntegrations();

  const integration = await getVercelIntegration();
  if (integration) {
    allIntegrations = [
      {
        slug: 'vercel',
        enabled: true,
        projects: integration.projects,
      },
      ...allIntegrations,
    ];
  }

  return <IntegrationsList integrations={allIntegrations} />;
}
