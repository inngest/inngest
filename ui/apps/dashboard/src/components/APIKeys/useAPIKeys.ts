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
        scopes {
          name
          allow
          deny
        }
      }
    }
  }
`);

const queryContext = { additionalTypenames: ['APIKey'] };

export function useAPIKeys(args: { workspaceID?: string } = {}) {
  return useGraphQLQuery({
    query: Query,
    variables: { workspaceID: args.workspaceID ?? null },
    context: queryContext,
  });
}
