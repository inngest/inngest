import { createFileRoute, redirect } from '@tanstack/react-router';
import IntegrationsPage from '@inngest/components/PostgresIntegrations/IntegrationPage';
import { integrationPageContent } from '@inngest/components/PostgresIntegrations/Supabase/newSupabaseContent';

import {
  deleteConn,
  PostgresIntegrations,
} from '@/queries/server/integrations/db';

export const Route = createFileRoute(
  '/_authed/settings/integrations/supabase/',
)({
  component: SupabasePage,
  loader: async () => {
    const postgresIntegrations = await PostgresIntegrations();
    const conns = postgresIntegrations.filter(
      (connection) => connection.slug === 'supabase',
    );

    if (conns.length === 0) {
      throw redirect({
        to: '/settings/integrations/supabase/connect',
      });
    }

    return { conns };
  },
});

function SupabasePage() {
  const { conns } = Route.useLoaderData();

  const handleDelete = async (id: string) => {
    try {
      await deleteConn({ data: { id } });
      return { success: true, error: null };
    } catch (error) {
      console.error('Error deleting connection:', error);
      return {
        success: false,
        error: 'Error removing Supabase integration, please try again later.',
      };
    }
  };

  return (
    <IntegrationsPage
      publications={conns}
      content={integrationPageContent}
      onDelete={handleDelete}
    />
  );
}
