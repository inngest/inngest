import { vercelIntegration } from './data';
import IntegrationsList from './integrations';

export default async function IntegrationsPage() {
  const integration = await vercelIntegration();

  return <IntegrationsList integration={integration} />;
}
