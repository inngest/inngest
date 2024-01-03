import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query AppSyncs($envID: ID!, $externalAppID: String!) {
    environment: workspace(id: $envID) {
      app: appByExternalID(externalID: $externalAppID) {
        id
        syncs(first: 40) {
          commitAuthor
          commitHash
          commitMessage
          commitRef
          createdAt
          framework
          id
          platform
          removedFunctions {
            id
            name
            slug
          }
          repoURL
          sdkLanguage
          sdkVersion
          status
          syncedFunctions: deployedFunctions {
            id
            name
            slug
          }
          url
          vercelDeploymentID
          vercelDeploymentURL
          vercelProjectID
          vercelProjectURL
        }
      }
    }
  }
`);

export function useSyncs({ envID, externalAppID }: { envID: string; externalAppID: string }) {
  const res = useGraphQLQuery({
    // pollIntervalInMilliseconds: 1000,
    query,
    variables: { envID, externalAppID },
  });

  if (res.data) {
    const { app } = res.data.environment;

    const syncs = app.syncs.map((sync) => {
      return {
        ...sync,
        createdAt: new Date(sync.createdAt),
      };
    });

    return {
      ...res,
      data: {
        ...app,
        syncs,
      },
    };
  }

  return res;
}
