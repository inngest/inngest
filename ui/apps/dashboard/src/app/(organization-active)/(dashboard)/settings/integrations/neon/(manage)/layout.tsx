import { redirect } from 'next/navigation';

import { PostgresIntegrations } from '@/components/PostgresIntegration/data';
import Page from './page';

export default async function Layout() {
  const postgresIntegrations = await PostgresIntegrations();
  const neonConnection = postgresIntegrations.find((connection) => connection.slug === 'neon');

  if (!neonConnection) {
    redirect('/settings/integrations/neon/connect');
  }

  return <Page publication={neonConnection} />;
}
