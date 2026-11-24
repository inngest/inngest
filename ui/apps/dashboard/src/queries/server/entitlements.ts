import { ClientError } from 'graphql-request';
import { graphql } from '@/gql';
import type { SandboxEntitlementsQuery } from '@/gql/graphql';
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
  let account: SandboxEntitlementsQuery['account'];
  try {
    ({ account } = await graphqlAPI.request(sandboxEntitlementsDocument));
  } catch (error) {
    // A failed verification read is a field error next to partial data. Keep
    // the entitlement, and let allowsSandboxList fall back to it.
    const data =
      error instanceof ClientError
        ? (error.response.data as SandboxEntitlementsQuery | undefined)
        : undefined;
    if (!data?.account) throw error;
    account = data.account;
  }

  return allowsSandboxList(
    account.computeVerification?.status,
    account.entitlements.sandboxes.enabled,
  );
});
