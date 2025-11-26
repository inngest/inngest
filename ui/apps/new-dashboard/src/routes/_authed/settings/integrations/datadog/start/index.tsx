import { createFileRoute } from "@tanstack/react-router";

import DatadogSetupPage from "@/components/Settings/DatadogSetupPage";
import StartPage from "@/components/DatadogIntegration/StartPage";
import { MetricsEntitlements } from "@/queries/server-only/entitlements";

type DatadogStartSearch = {
  site?: string;
  domain?: string;
};

export const Route = createFileRoute(
  "/_authed/settings/integrations/datadog/start/",
)({
  component: DatadogStartPage,
  validateSearch: (search: Record<string, unknown>): DatadogStartSearch => ({
    site: search.site as string | undefined,
    domain: search.domain as string | undefined,
  }),
  loader: async () => {
    const metricsEntitlements = await MetricsEntitlements();
    return { metricsEntitlements };
  },
});

function DatadogStartPage() {
  const { metricsEntitlements } = Route.useLoaderData();
  const { site, domain } = Route.useSearch();

  return (
    <DatadogSetupPage
      title="Connect to Datadog"
      showSubtitleDocsLink={false}
      content={<StartPage site={site} domain={domain} />}
      metricsEntitlements={metricsEntitlements}
    />
  );
}
