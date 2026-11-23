import { graphql } from '@/gql';
import { allowsSandboxList } from '@/components/ComputeVerification/adapter';
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
      computeVerification { status }
      entitlements: ents {
        sandboxes {
          enabled
        }
      }
    }
  }
`);

// UI access: verification takes precedence; legacy accounts retain their entitlement gate.
export const getSandboxesEnabled = createServerFn({
  method: 'GET',
}).handler(async () => {
  const response = await graphqlAPI.request(sandboxEntitlementsDocument);

  return allowsSandboxList(
    response.account.computeVerification?.status,
    response.account.entitlements.sandboxes.enabled,
  );
});
