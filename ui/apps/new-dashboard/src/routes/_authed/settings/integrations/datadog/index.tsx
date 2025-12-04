import { createFileRoute } from "@tanstack/react-router";

import DatadogSetupPage from "@/components/Settings/DatadogSetupPage";
import SetupPage from "@/components/DatadogIntegration/SetupPage";
import { MetricsEntitlements } from "@/queries/server/entitlements";

export const Route = createFileRoute("/_authed/settings/integrations/datadog/")(
  {
    component: DatadogPage,
    loader: async () => {
      const metricsEntitlements = await MetricsEntitlements();
      return { metricsEntitlements };
    },
  },
);

function DatadogPage() {
  const { metricsEntitlements } = Route.useLoaderData();

  return (
    <DatadogSetupPage
      subtitle="Send key Inngest metrics directly to your Datadog account."
      showEntitlements={true}
      content={<SetupPage />}
      metricsEntitlements={metricsEntitlements}
    />
  );
}
