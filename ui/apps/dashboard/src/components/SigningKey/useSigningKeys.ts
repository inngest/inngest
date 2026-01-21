import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const Query = graphql(`
  query GetSigningKeys($envID: ID!) {
    environment: workspace(id: $envID) {
      signingKeys {
        createdAt
        decryptedValue
        id
        isActive
        user {
          email
          name
        }
      }
    }
  }
`);

export function useSigningKeys({ envID }: { envID: string }) {
  const res = useGraphQLQuery({
    query: Query,
    variables: { envID },
  });

  if (!res.data) {
    return res;
  }

  return {
    ...res,
    data: res.data.environment.signingKeys.map((key) => {
      return {
        ...key,
        createdAt: new Date(key.createdAt),
      };
    }),
  };
}
