import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { workspacesToEnvironments, type Environment } from '@/utils/environments';

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

export async function getEnvironment({
  environmentSlug,
}: GetEnvironmentParams): Promise<Environment> {
  const query = await graphqlAPI.request(GetEnvironmentBySlugDocument, {
    slug: environmentSlug,
  });
  if (!query.envBySlug) {
    throw new Error(`Environment "${environmentSlug}" not found`);
  }

  const environment = workspacesToEnvironments([query.envBySlug])[0];
  if (!environment) {
    throw new Error(`Failed to convert workspace "${environmentSlug}" to environment`);
  }

  return environment;
}

export async function getProductionEnvironment(): Promise<Environment> {
  const query = await graphqlAPI.request(GetProductionWorkspaceDocument);

  const environment = workspacesToEnvironments([query.defaultEnv])[0];
  if (!environment) {
    throw new Error(`Failed to convert production workspace to environment`);
  }

  return environment;
}
