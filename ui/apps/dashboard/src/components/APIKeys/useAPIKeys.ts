import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const Query = graphql(`
  query GetAPIKeys($workspaceID: UUID) {
    account {
      apiKeys(workspaceID: $workspaceID) {
        id
        name
        createdAt
        maskedKey
        workspace {
          id
          name
          slug
        }
      }
    }
  }
`);

const queryContext = { additionalTypenames: ['APIKey'] };

export function useAPIKeys(args: { workspaceID?: string } = {}) {
  return useGraphQLQuery({
    query: Query,
    // null = "all workspaces in the account"; the backend treats an omitted
    // workspaceID the same as an explicit null.
    variables: { workspaceID: args.workspaceID ?? null },
    context: queryContext,
  });
}
