import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query UnattachedSync($syncID: ID!) {
    sync: deploy(id: $syncID) {
      commitAuthor
      commitHash
      commitMessage
      commitRef
      error
      framework
      id
      lastSyncedAt
      platform
      repoURL
      sdkLanguage
      sdkVersion
      status
      removedFunctions: removedFunctions {
        id
        name
        slug
      }
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
`);

export function useSync({ syncID }: { syncID: string }) {
  const res = useGraphQLQuery({
    query,
    variables: { syncID },
  });

  if (res.data) {
    const sync = {
      ...res.data.sync,
      lastSyncedAt: new Date(res.data.sync.lastSyncedAt),
    };

    return {
      ...res,
      data: sync,
    };
  }

  return res;
}
