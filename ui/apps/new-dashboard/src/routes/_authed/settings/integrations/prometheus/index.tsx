import { createFileRoute } from "@tanstack/react-router";

import SetupPage from "@/components/PrometheusIntegration/SetupPage";
import { MetricsEntitlements } from "@/queries/server-only/entitlements";

export const Route = createFileRoute(
  "/_authed/settings/integrations/prometheus/",
)({
  component: PrometheusPage,
  loader: async () => {
    const metricsEntitlements = await MetricsEntitlements();
    return {
      metricsEntitlements,
    };
  },
});

function PrometheusPage() {
  const { metricsEntitlements } = Route.useLoaderData();

  return (
    <SetupPage
      metricsExportEnabled={metricsEntitlements.metricsExport.enabled}
      metricsGranularitySeconds={
        metricsEntitlements.metricsExportGranularity.limit
      }
      metricsFreshnessSeconds={metricsEntitlements.metricsExportFreshness.limit}
    />
  );
}
