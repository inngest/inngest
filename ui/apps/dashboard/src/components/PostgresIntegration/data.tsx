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

    console.log(integrations);

    return integrations.map((integration) => {
      // The DB name has a prefix, eg "Neon-" or "Supabase-" which is the slug.  This dictates which
      // "integration" (postgres host) was used to set up the connection.
      const slug = (integration.name.split('-')[0] || '').toLowerCase();

      return {
        id: integration.id,
        name: integration.name,
        slug,
        projects: [],
        enabled: integration.status === 'RUNNING' || integration.status === 'SETUP_COMPLETE',
      };
    });
  } catch (error) {
    return [];
  }
};
