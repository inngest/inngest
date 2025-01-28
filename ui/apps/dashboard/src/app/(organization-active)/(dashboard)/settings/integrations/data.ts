import { graphql } from '@/gql/gql';
import type { VercelIntegration } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';

const vercelIntegrationQuery = graphql(`
  query VercelIntegration {
    account {
      vercelIntegration {
        isMarketplace
        projects {
          canChangeEnabled
          deploymentProtection
          isEnabled
          name
          originOverride
          projectID
          protectionBypassSecret
          servePath
        }
      }
    }
  }
`);

export async function getVercelIntegration(): Promise<VercelIntegration | null> {
  try {
    const res = await graphqlAPI.request(vercelIntegrationQuery);
    return res.account.vercelIntegration ?? null;
  } catch (err) {
    // TODO: Handle this in the backend instead of swallowing here.
    console.error(err);
    return null;
  }
}
