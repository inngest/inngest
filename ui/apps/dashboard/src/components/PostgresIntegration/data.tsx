import { graphql } from '@/gql';
import { type CdcConnection } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getProductionEnvironment } from '@/queries/server-only/getEnvironment';

const getPostgresIntegrationsDocument = graphql(`
  query getPostgresIntegrations($envID: ID!) {
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

export const PostgresIntegrations = async () => {
  try {
    const environment = await getProductionEnvironment();
    const response = await graphqlAPI.request<{ environment: { cdcConnections: CdcConnection[] } }>(
      getPostgresIntegrationsDocument,
      { envID: environment.id }
    );

    const integrations = response.environment.cdcConnections;

    return integrations.map((integration) => ({
      id: integration.id,
      name: integration.name,
      slug: 'neon',
      projects: [],
      enabled: integration.status === 'RUNNING' || integration.status === 'SETUP_COMPLETE',
    }));
  } catch (error) {
    return [];
  }
};
