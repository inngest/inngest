import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query GetEventKeysForBlankSlate($environmentID: ID!) {
    environment: workspace(id: $environmentID) {
      ingestKeys(filter: { source: "key" }) {
        name
        presharedKey
        createdAt
      }
    }
  }
`);

export function useDefaultEventKey({ envID }: { envID: string }) {
  const res = useGraphQLQuery({
    query,
    variables: { environmentID: envID },
  });

  if (res.data) {
    const keys = res.data.environment.ingestKeys;
    const defaultKey = getDefaultEventKey(keys);

    if (!defaultKey) {
      throw new Error(`No default key found in ${keys}`);
    }

    return {
      ...res,
      data: {
        defaultKey,
      },
    };
  }

  return {
    ...res,
    data: undefined,
  };
}

function getDefaultEventKey<T extends { createdAt: string; name: null | string }>(
  keys: T[]
): T | undefined {
  const def = keys.find((k) => k.name && k.name.match(/default ingest/i));

  return (
    def ||
    [...keys].sort((a, b) => {
      return Date.parse(a.createdAt) - Date.parse(b.createdAt);
    })[0]
  );
}
