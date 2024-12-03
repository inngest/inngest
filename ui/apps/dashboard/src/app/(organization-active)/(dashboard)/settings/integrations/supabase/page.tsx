import { redirect } from 'next/navigation';
import IntegrationsPage from '@inngest/components/PostgresIntegrations/IntegrationPage';
import { integrationPageContent } from '@inngest/components/PostgresIntegrations/Supabase/supabaseContent';

import { PostgresIntegrations } from '@/components/PostgresIntegration/data';

export default async function Page() {
  const postgresIntegrations = await PostgresIntegrations();

  console.log('found', postgresIntegrations);

  const conn = postgresIntegrations.find((connection) => connection.slug === 'supabase');

  if (!conn) {
    redirect('/settings/integrations/supabase/connect');
  }

  return <IntegrationsPage publications={[conn]} content={integrationPageContent} />;
}
