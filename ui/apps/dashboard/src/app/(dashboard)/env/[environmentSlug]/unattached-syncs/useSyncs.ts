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
        createdAt
        framework
        id
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
        createdAt: new Date(sync.createdAt),
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
