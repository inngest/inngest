import { graphql } from "@/gql";
import type { CdcConnection } from "@/gql/graphql";
import graphqlAPI from "@/queries/graphqlAPI";
import { getProductionEnvironment } from "@/queries/server-only/getEnvironment";
import { createServerFn } from "@tanstack/react-start";

import {
  type CdcConnectionInput,
  type CdcSetupResponse,
  type DeleteResponse,
} from "@/gql/graphql";

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

const testAuthDocument = graphql(`
  mutation testCredentials($input: CDCConnectionInput!, $envID: UUID!) {
    cdcTestCredentials(input: $input, envID: $envID) {
      steps
      error
    }
  }
`);

const testAuth = async (input: CdcConnectionInput) => {
  const environment = await getProductionEnvironment();

  return await graphqlAPI.request<{ cdcTestCredentials: CdcSetupResponse }>(
    testAuthDocument,
    {
      envID: environment.id,
      input: input,
    },
  );
};

const testLogicalReplicationDocument = graphql(`
  mutation testReplication($input: CDCConnectionInput!, $envID: UUID!) {
    cdcTestLogicalReplication(input: $input, envID: $envID) {
      steps
      error
    }
  }
`);

const testLogicalReplication = async (input: CdcConnectionInput) => {
  const environment = await getProductionEnvironment();

  return await graphqlAPI.request<{
    cdcTestLogicalReplication: CdcSetupResponse;
  }>(testLogicalReplicationDocument, {
    envID: environment.id,
    input: input,
  });
};

const testAutoSetupDocument = graphql(`
  mutation testAutoSetup($input: CDCConnectionInput!, $envID: UUID!) {
    cdcAutoSetup(input: $input, envID: $envID) {
      steps
      error
    }
  }
`);

const testAutoSetup = async (input: CdcConnectionInput) => {
  const environment = await getProductionEnvironment();

  return await graphqlAPI.request<{ cdcAutoSetup: CdcSetupResponse }>(
    testAutoSetupDocument,
    {
      envID: environment.id,
      input: input,
    },
  );
};

const deleteConnDocument = graphql(`
  mutation cdcDelete($envID: UUID!, $id: UUID!) {
    cdcDelete(envID: $envID, id: $id) {
      ids
    }
  }
`);

export const deleteConn = createServerFn({ method: "POST" })
  .inputValidator((data: { id: string }) => data)
  .handler(async ({ data }) => {
    const environment = await getProductionEnvironment();

    return await graphqlAPI.request<{ cdcDelete: DeleteResponse }>(
      deleteConnDocument as any,
      {
        envID: environment.id,
        id: data.id,
      },
    );
  });

export const verifyCredentials = createServerFn({ method: "POST" })
  .inputValidator((data: { input: CdcConnectionInput }) => data)
  .handler(async ({ data }) => {
    try {
      const response = await testAuth(data.input);
      const error = response.cdcTestCredentials.error;
      if (error) {
        return { success: false, error: error };
      }
      return { success: true, error: null };
    } catch (error) {
      console.error("Error verifying credentials:", error);
      return { success: false, error: null };
    }
  });

export const verifyLogicalReplication = createServerFn({ method: "POST" })
  .inputValidator((data: { input: CdcConnectionInput }) => data)
  .handler(async ({ data }) => {
    try {
      const response = await testLogicalReplication(data.input);
      const error = response.cdcTestLogicalReplication.error;
      if (error) {
        return { success: false, error: error };
      }
      return { success: true, error: null };
    } catch (error) {
      console.error("Error verifying logical replication:", error);
      return { success: false, error: null };
    }
  });

export const verifyAutoSetup = createServerFn({ method: "POST" })
  .inputValidator((data: { input: CdcConnectionInput }) => data)
  // @ts-expect-error - TANSTACK TODO: sort out type issue
  .handler(async ({ data }) => {
    try {
      const response = await testAutoSetup(data.input);
      const error = response.cdcAutoSetup.error;
      if (error) {
        return {
          success: false,
          error: error,
          steps: response.cdcAutoSetup.steps,
        };
      }
      return { success: true, error: null, steps: response.cdcAutoSetup.steps };
    } catch (error) {
      console.error("Error connecting:", error);
      return { success: false, error: null };
    }
  });
