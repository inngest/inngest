import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { createServerFn } from '@tanstack/react-start';

const metricsEntitlementsDocument = graphql(`
  query MetricsEntitlements {
    account {
      id
      entitlements: ents {
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

const sandboxEntitlementsDocument = graphql(`
  query SandboxEntitlements {
    account {
      id
      entitlements: ents {
        sandboxes {
          enabled
        }
      }
    }
  }
`);

// Whether the account's current plan grants sandbox access.
export const getSandboxesEnabled = createServerFn({
  method: 'GET',
}).handler(async () => {
  const response = await graphqlAPI.request(sandboxEntitlementsDocument);

  return response.account.entitlements.sandboxes.enabled;
});
