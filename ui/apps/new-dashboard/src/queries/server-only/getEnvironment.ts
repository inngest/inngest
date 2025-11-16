import { graphql } from "@/gql";

import { workspacesToEnvironments } from "@/utils/environments";
import { createServerFn } from "@tanstack/react-start";
import { graphqlAPI } from "../graphqlAPI";

const GetEnvironmentBySlugDocument = graphql(`
  query GetEnvironmentBySlug($slug: String!) {
    envBySlug(slug: $slug) {
      id
      name
      slug
      parentID
      test
      type
      createdAt
      lastDeployedAt
      isArchived
      isAutoArchiveEnabled
      webhookSigningKey
    }
  }
`);

const GetProductionWorkspaceDocument = graphql(`
  query GetProductionWorkspace {
    defaultEnv {
      id
      name
      slug
      parentID
      test
      type
      createdAt
      lastDeployedAt
      isArchived
      isAutoArchiveEnabled
      webhookSigningKey
    }
  }
`);

type GetEnvironmentParams = {
  environmentSlug: string;
};

export const getEnvironment = createServerFn({
  method: "GET",
})
  .inputValidator((data: GetEnvironmentParams) => data)
  .handler(async ({ data }) => {
    const { environmentSlug } = data;
    const query = await graphqlAPI.request(GetEnvironmentBySlugDocument, {
      slug: environmentSlug,
    });
    if (!query.envBySlug) {
      throw new Error(`Environment "${environmentSlug}" not found`);
    }

    const environment = workspacesToEnvironments([query.envBySlug])[0];
    if (!environment) {
      throw new Error(
        `Failed to convert workspace "${environmentSlug}" to environment`,
      );
    }

    return environment;
  });

export const getProductionEnvironment = createServerFn({
  method: "GET",
}).handler(async () => {
  const query = await graphqlAPI.request(GetProductionWorkspaceDocument);
  const environment = workspacesToEnvironments([query.defaultEnv])[0];
  if (!environment) {
    throw new Error(`Failed to convert production workspace to environment`);
  }
  return environment;
});
