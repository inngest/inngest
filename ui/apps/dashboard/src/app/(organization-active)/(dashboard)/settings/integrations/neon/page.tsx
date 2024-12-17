import { redirect } from 'next/navigation';
import IntegrationsPage from '@inngest/components/PostgresIntegrations/IntegrationPage';
import { neonIntegrationPageContent } from '@inngest/components/PostgresIntegrations/Neon/neonContent';

import { PostgresIntegrations } from '@/components/PostgresIntegration/data';
import { deleteConn } from '@/components/PostgresIntegration/neonData';

export default async function Page() {
  const postgresIntegrations = await PostgresIntegrations();
  const neonConnection = postgresIntegrations.find((connection) => connection.slug === 'neon');

  if (!neonConnection) {
    redirect('/settings/integrations/neon/connect');
  }

  const onDelete = async () => {
    await deleteConn(neonConnection.id);
    redirect('/settings/integrations/neon');
  };

  return (
    <IntegrationsPage
      publications={[neonConnection]}
      content={neonIntegrationPageContent}
      onDelete={onDelete}
    />
  );
}
