import { graphql } from '@/gql';
import type { Workspace } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';
import { workspacesToEnvironments, type Environment } from '@/utils/environments';

const GetAllEnvironmentsDocument = graphql(`
  query GetAllEnvironments {
    workspaces {
      id
      name
      parentID
      test
      type
      createdAt
      isArchived
      isAutoArchiveEnabled
    }
  }
`);

export default async function getAllEnvironments(): Promise<Environment[]> {
  const { workspaces } = await graphqlAPI.request(GetAllEnvironmentsDocument);
  return workspacesToEnvironments(workspaces as Workspace[]);
}
