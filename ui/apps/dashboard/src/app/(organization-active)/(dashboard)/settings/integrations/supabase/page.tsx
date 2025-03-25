import { redirect } from 'next/navigation';
import IntegrationsPage from '@inngest/components/PostgresIntegrations/IntegrationPage';
import { integrationPageContent } from '@inngest/components/PostgresIntegrations/Supabase/supabaseContent';

import { PostgresIntegrations } from '@/components/PostgresIntegration/data';
import { deleteConn } from '@/components/PostgresIntegration/neonData';

export default async function Page() {
  const postgresIntegrations = await PostgresIntegrations();

  const conn = postgresIntegrations.find((connection) => connection.slug === 'supabase');

  if (!conn) {
    redirect('/settings/integrations/supabase/connect');
  }

  const onDelete = async () => {
    await deleteConn(conn.id);
    redirect('/settings/integrations/supabase');
  };

  return (
    <IntegrationsPage publications={[conn]} content={integrationPageContent} onDelete={onDelete} />
  );
}
