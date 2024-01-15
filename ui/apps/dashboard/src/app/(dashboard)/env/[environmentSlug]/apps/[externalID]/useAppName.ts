import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query AppName($envID: ID!, $externalAppID: String!) {
    environment: workspace(id: $envID) {
      app: appByExternalID(externalID: $externalAppID) {
        name
      }
    }
  }
`);

export function useAppName({ envID, externalAppID }: { envID: string; externalAppID: string }) {
  const res = useGraphQLQuery({
    query,
    variables: { envID, externalAppID },
  });

  if (res.data) {
    return {
      ...res,
      data: res.data.environment.app.name,
    };
  }

  return res;
}
