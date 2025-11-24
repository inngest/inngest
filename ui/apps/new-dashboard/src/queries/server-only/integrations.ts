import { graphql } from "@/gql";
import { type CdcConnection } from "@/gql/graphql";
import graphqlAPI from "@/queries/graphqlAPI";
import { getProductionEnvironment } from "@/queries/server-only/getEnvironment";
import { createServerFn } from "@tanstack/react-start";

import { ClientError } from "graphql-request";
import type { VercelIntegration } from "@/gql/graphql";

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

export const PostgresIntegrations = createServerFn({
  method: "GET",
}).handler(async () => {
  try {
    const environment = await getProductionEnvironment();
    const response = await graphqlAPI.request<{
      environment: { cdcConnections: CdcConnection[] };
    }>(getPostgresIntegrationsDocument, { envID: environment.id });

    const integrations = response.environment.cdcConnections;

    return integrations.map((integration) => {
      // The DB name has a prefix, eg "Neon-" or "Supabase-" which is the slug.  This dictates which
      // "integration" (postgres host) was used to set up the connection.
      const slug = (integration.name.split("-")[0] || "").toLowerCase();

      return {
        id: integration.id,
        name: integration.name,
        slug,
        projects: [],
        enabled:
          integration.status === "RUNNING" ||
          integration.status === "SETUP_COMPLETE",
      };
    });
  } catch (error) {
    return [];
  }
});

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

export const getVercelIntegration = createServerFn({
  method: "GET",
  //@ts-expect-error TANSTACK TODO: sort out type error here
}).handler(async () => {
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
});
