import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';

const metricsEntitlementsDocument = graphql(`
  query MetricsEntitlements {
    account {
      id
      entitlements {
        metricsExport {
          enabled
        }
        metricsExportFreshness {
          limit
        }
        metricsExportGranularity {
          limit
        }
      }
    }
  }
`);

export const MetricsEntitlements = async () => {
  const response = await graphqlAPI.request(metricsEntitlementsDocument);

  return response.account.entitlements;
};
