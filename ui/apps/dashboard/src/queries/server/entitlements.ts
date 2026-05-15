import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { createServerFn } from '@tanstack/react-start';

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

export const MetricsEntitlements = createServerFn({
  method: 'GET',
}).handler(async () => {
  const response = await graphqlAPI.request(metricsEntitlementsDocument);

  return response.account.entitlements;
});
