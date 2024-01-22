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
          framework
          id
          lastSyncedAt
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
    query,
    variables: { envID, externalAppID },
  });

  if (res.data) {
    const { app } = res.data.environment;

    const syncs = app.syncs.map((sync) => {
      return {
        ...sync,
        lastSyncedAt: new Date(sync.lastSyncedAt),
      };
    });

    return {
      ...res,
      data: syncs,
    };
  }

  return res;
}
