import { redirect } from 'next/navigation';

import { PostgresIntegrations } from '@/components/PostgresIntegration/data';
import Manage from './manage';

export default async function Layout() {
  const postgresIntegrations = await PostgresIntegrations();
  const neonConnection = postgresIntegrations.find((connection) => connection.slug === 'neon');

  if (!neonConnection) {
    redirect('/settings/integrations/neon/connect');
  }

  return <Manage publication={neonConnection} />;
}
