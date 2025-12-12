import { createFileRoute, redirect } from '@tanstack/react-router';

import { PostgresIntegrations } from '@/queries/server/integrations/db';
import Manage from '@/components/PostgresIntegration/Manage';

export const Route = createFileRoute('/_authed/settings/integrations/neon/')({
  component: NeonManagePage,
  loader: async () => {
    const postgresIntegrations = await PostgresIntegrations();
    const neonConnection = postgresIntegrations.find(
      (connection) => connection.slug === 'neon',
    );

    if (!neonConnection) {
      throw redirect({
        to: '/settings/integrations/neon/connect',
      });
    }

    return { neonConnection };
  },
});

function NeonManagePage() {
  const { neonConnection } = Route.useLoaderData();

  return <Manage publication={neonConnection} />;
}
