import { ClientError } from "graphql-request";

import { graphql } from "@/gql/gql";
import type { VercelIntegration } from "@/gql/graphql";
import graphqlAPI from "@/queries/graphqlAPI";

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

export async function getVercelIntegration(): Promise<
  VercelIntegration | null | Error
> {
  try {
    const res = await graphqlAPI.request(vercelIntegrationQuery);
    return res.account.vercelIntegration ?? null;
  } catch (err) {
    if (err instanceof ClientError) {
      return new Error(err.response.errors?.[0]?.message ?? "Unknown error");
    }
    if (err instanceof Error) {
      return err;
    }
    return new Error("Unknown error");
  }
}
