import { createFileRoute } from '@tanstack/react-router';

import DatadogSetupPage from '@/components/Settings/DatadogSetupPage';
import AddConnectionPage from '@/components/DatadogIntegration/AddConnectionPage';
import { MetricsEntitlements } from '@/queries/server/entitlements';

export const Route = createFileRoute(
  '/_authed/settings/integrations/datadog/connect-env/',
)({
  component: DatadogConnectEnvPage,
  loader: async () => {
    const metricsEntitlements = await MetricsEntitlements();
    return { metricsEntitlements };
  },
});

function DatadogConnectEnvPage() {
  const { metricsEntitlements } = Route.useLoaderData();

  return (
    <DatadogSetupPage
      subtitle="Connect an environment to Datadog to send key metrics from Inngest to your Datadog account."
      content={<AddConnectionPage />}
      metricsEntitlements={metricsEntitlements}
    />
  );
}
