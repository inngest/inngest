import { PostgresIntegrations } from '@/components/PostgresIntegration/data';
import { vercelIntegration } from './data';
import IntegrationsList from './integrations';

export default async function IntegrationsPage() {
  const integration = await vercelIntegration();
  const postgresIntegrations = await PostgresIntegrations();

  const allIntegrations = [integration, ...postgresIntegrations];

  return <IntegrationsList integrations={allIntegrations} />;
}
