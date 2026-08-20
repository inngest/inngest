import { gql, type TypedDocumentNode } from 'urql';

import { ApiKeyOwnershipType } from '@/gql/graphql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

type GetAPIKeysResult = {
  [key: string]: unknown;
  session: {
    user: {
      id: string;
    };
  } | null;
  account: {
    apiKeys: {
      id: string;
      name: string;
      createdAt: string;
      ownershipType: ApiKeyOwnershipType;
      ownerUserID: string | null;
      maskedKey: string;
      env: {
        id: string;
        name: string;
        slug: string;
      } | null;
    }[];
  };
};

type GetAPIKeysVariables = {
  [key: string]: unknown;
  workspaceID: string | null;
};

const Query: TypedDocumentNode<GetAPIKeysResult, GetAPIKeysVariables> = gql`
  query GetAPIKeys($workspaceID: UUID) {
    session {
      user {
        id
      }
    }
    account {
      apiKeys(workspaceID: $workspaceID) {
        id
        name
        createdAt
        ownershipType
        ownerUserID
        maskedKey
        env {
          id
          name
          slug
        }
      }
    }
  }
`;

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
