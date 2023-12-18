import { graphql } from '@/gql';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query AppName($envID: ID!, $externalAppID: String!) {
    environment: workspace(id: $envID) {
      app: appByExternalID(externalID: $externalAppID) {
        name
      }
    }
  }
`);

export function useAppName({
  envID,
  externalAppID,
  skip,
}: {
  envID: string;
  externalAppID: string;
  skip: boolean;
}) {
  const res = useSkippableGraphQLQuery({
    query,
    skip,
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
