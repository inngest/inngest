import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query UnattachedSync($syncID: ID!) {
    sync: deploy(id: $syncID) {
      commitAuthor
      commitHash
      commitMessage
      commitRef
      createdAt
      framework
      id
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
      createdAt: new Date(res.data.sync.createdAt),
    };

    return {
      ...res,
      data: sync,
    };
  }

  return res;
}
