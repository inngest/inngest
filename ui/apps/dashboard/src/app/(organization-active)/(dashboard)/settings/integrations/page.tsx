import { PostgresIntegrations } from "@/components/PostgresIntegration/data";
import { getVercelIntegration } from "./data";
import IntegrationsList from "./integrations";

export default async function IntegrationsPage() {
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

  return <IntegrationsList integrations={allIntegrations} />;
}
