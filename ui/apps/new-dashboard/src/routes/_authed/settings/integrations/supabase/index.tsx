import { createFileRoute, redirect } from "@tanstack/react-router";
import IntegrationsPage from "@inngest/components/PostgresIntegrations/NewIntegrationPage";
import { integrationPageContent } from "@inngest/components/PostgresIntegrations/Supabase/newSupabaseContent";

import { PostgresIntegrations } from "@/queries/server-only/integrations/db";
import { deleteConn } from "@/queries/server-only/integrations/db";

export const Route = createFileRoute(
  "/_authed/settings/integrations/supabase/",
)({
  component: SupabasePage,
  loader: async () => {
    const postgresIntegrations = await PostgresIntegrations();
    const conn = postgresIntegrations.find(
      (connection) => connection.slug === "supabase",
    );

    if (!conn) {
      throw redirect({
        to: "/settings/integrations/supabase/connect",
      });
    }

    return { conn };
  },
});

function SupabasePage() {
  const { conn } = Route.useLoaderData();

  const handleDelete = async (id: string) => {
    try {
      await deleteConn({ data: { id } });
      return { success: true, error: null };
    } catch (error) {
      console.error("Error deleting connection:", error);
      return {
        success: false,
        error: "Error removing Supabase integration, please try again later.",
      };
    }
  };

  return (
    <IntegrationsPage
      publications={[conn]}
      content={integrationPageContent}
      onDelete={handleDelete}
    />
  );
}
