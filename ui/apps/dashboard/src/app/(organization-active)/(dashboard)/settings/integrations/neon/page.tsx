import { redirect } from 'next/navigation';
import IntegrationsPage from '@inngest/components/PostgresIntegrations/IntegrationPage';
import { neonIntegrationPageContent } from '@inngest/components/PostgresIntegrations/Neon/neonContent';

import { graphql } from '@/gql';
import { type CdcConnection } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getProductionEnvironment } from '@/queries/server-only/getEnvironment';

const getConnectionDocument = graphql(`
  query getNeon($envID: ID!) {
    environment: workspace(id: $envID) {
      cdcConnections {
        id
        name
        status
        statusDetail
        description
      }
    }
  }
`);

export default async function Page() {
  const environment = await getProductionEnvironment();
  const response = await graphqlAPI.request<{ environment: { cdcConnections: CdcConnection[] } }>(
    getConnectionDocument,
    { envID: environment.id }
  );

  const integrations = response.environment.cdcConnections;
  const neonConnection = integrations.find((connection) => connection.name.startsWith('Neon'));

  if (!neonConnection) {
    redirect('/settings/integrations/neon/connect');
  }

  const publications = [
    {
      isActive: neonConnection.status === 'RUNNING',
      name: neonConnection.name,
    },
  ];

  return <IntegrationsPage publications={publications} content={neonIntegrationPageContent} />;
}
