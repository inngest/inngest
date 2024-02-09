import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query UnattachedSyncs($envID: ID!) {
    environment: workspace(id: $envID) {
      syncs: unattachedSyncs(first: 40) {
        commitAuthor
        commitHash
        commitMessage
        commitRef
        framework
        id
        lastSyncedAt
        platform
        repoURL
        sdkLanguage
        sdkVersion
        status
        url
        vercelDeploymentID
        vercelDeploymentURL
        vercelProjectID
        vercelProjectURL
      }
    }
  }
`);

export function useSyncs({ envID }: { envID: string }) {
  const res = useGraphQLQuery({
    query,
    variables: { envID },
  });

  if (res.data) {
    const syncs = res.data.environment.syncs.map((sync) => {
      return {
        ...sync,
        lastSyncedAt: new Date(sync.lastSyncedAt),
        syncedFunctions: [],
      };
    });

    return {
      ...res,
      data: syncs,
    };
  }

  return res;
}
