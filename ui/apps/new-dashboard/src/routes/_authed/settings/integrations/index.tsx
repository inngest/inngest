import { createFileRoute } from "@tanstack/react-router";

import { IntegrationsList } from "@/components/Settings/IntegrationsList";
import { PostgresIntegrations } from "@/queries/server/integrations/db";
import { getVercelIntegration } from "@/queries/server/integrations/vercel";

export const Route = createFileRoute("/_authed/settings/integrations/")({
  component: IntegrationsPage,
  loader: async () => {
    let allIntegrations: React.ComponentProps<
      typeof IntegrationsList
    >["integrations"] = await PostgresIntegrations();

    const integration = await getVercelIntegration();
    if (integration) {
      if (integration instanceof Error) {
        allIntegrations = [
          {
            enabled: true,
            error: integration.message,
            projects: [],
            slug: "vercel",
          },
          ...allIntegrations,
        ];
      } else {
        allIntegrations = [
          {
            enabled: true,
            isMarketplace: integration.isMarketplace,
            projects: integration.projects,
            slug: "vercel",
          },
          ...allIntegrations,
        ];
      }
    }

    return { allIntegrations };
  },
});

function IntegrationsPage() {
  const { allIntegrations } = Route.useLoaderData();

  return <IntegrationsList integrations={allIntegrations} />;
}
