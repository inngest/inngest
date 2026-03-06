import { createFileRoute, redirect } from '@tanstack/react-router';

import { PostgresIntegrations } from '@/queries/server/integrations/db';
import Manage from '@/components/PostgresIntegration/Manage';

export const Route = createFileRoute('/_authed/settings/integrations/neon/')({
  component: NeonManagePage,
  loader: async () => {
    const postgresIntegrations = await PostgresIntegrations();
    const neonConnections = postgresIntegrations.filter(
      (connection) => connection.slug === 'neon',
    );

    if (neonConnections.length === 0) {
      throw redirect({
        to: '/settings/integrations/neon/connect',
      });
    }

    return { neonConnections };
  },
});

function NeonManagePage() {
  const { neonConnections } = Route.useLoaderData();

  return <Manage publications={neonConnections} />;
}
