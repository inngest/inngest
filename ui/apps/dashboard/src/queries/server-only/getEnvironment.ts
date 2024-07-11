import { graphql } from '@/gql';
import type { Workspace } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';
import { workspacesToEnvironments, type Environment } from '@/utils/environments';

const GetEnvironmentBySlugDocument = graphql(`
  query GetEnvironmentBySlug($slug: String!) {
    workspaceBySlug(slug: $slug) {
      id
      name
      parentID
      test
      type
      createdAt
      lastDeployedAt
      isArchived
      isAutoArchiveEnabled
    }
  }
`);

const GetProductionWorkspaceDocument = graphql(`
  query GetProductionWorkspace {
    productionWorkspace {
      id
      name
      parentID
      test
      type
      createdAt
      lastDeployedAt
      isArchived
      isAutoArchiveEnabled
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
  if (!query.workspaceBySlug) {
    throw new Error(`Environment "${environmentSlug}" not found`);
  }

  const environment = workspacesToEnvironments([query.workspaceBySlug] as Workspace[])[0];
  if (!environment) {
    throw new Error(`Failed to convert workspace "${environmentSlug}" to environment`);
  }

  return environment;
}

export async function getProductionEnvironment(): Promise<Environment> {
  const query = await graphqlAPI.request(GetProductionWorkspaceDocument);

  const environment = workspacesToEnvironments([query.productionWorkspace] as Workspace[])[0];
  if (!environment) {
    throw new Error(`Failed to convert production workspace to environment`);
  }

  return environment;
}
